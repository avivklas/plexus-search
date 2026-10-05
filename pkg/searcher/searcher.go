package searcher

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/avivklas/plexus"
	"github.com/avivklas/plexus-search/pkg/docstore"
	"github.com/avivklas/plexus-search/pkg/indexstore"
)

var (
	ErrNotFound      = docstore.ErrNotFound
	ErrMissingIndex  = docstore.ErrMissingIndex
	ErrNotLeader     = errors.New("node is not cluster leader")
)

// Searcher coordinates DocumentStore and IndexStore within a single Plexus node.
type Searcher struct {
	mu          sync.RWMutex
	docStore    *docstore.Store
	indexStore  *indexstore.Store
	coordinator *CoordinatorStore
	cluster     *plexus.Cluster
	mutator     plexus.Mutator
	startTime   time.Time
}

// New creates an initialized Searcher coordinator.
func New(ds *docstore.Store, is *indexstore.Store) *Searcher {
	return &Searcher{
		docStore:    ds,
		indexStore:  is,
		coordinator: NewCoordinatorStore(ds, is),
		startTime:   time.Now().UTC(),
	}
}

// Register registers all three stores (docstore, indexstore, coordinator) into the Plexus cluster
// and attaches the cluster mutator.
func (s *Searcher) Register(c *plexus.Cluster) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.cluster = c
	c.UseState(s.docStore)
	c.UseState(s.indexStore)
	mut := c.UseState(s.coordinator)

	s.mutator = mut
	s.docStore.AttachMutator(mut)
	s.indexStore.AttachMutator(mut)
}

// AttachMutator directly sets the cluster mutator.
func (s *Searcher) AttachMutator(m plexus.Mutator) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.mutator = m
	s.docStore.AttachMutator(m)
	s.indexStore.AttachMutator(m)
}

// DocStore returns the underlying docstore.Store.
func (s *Searcher) DocStore() *docstore.Store {
	return s.docStore
}

// IndexStore returns the underlying indexstore.Store.
func (s *Searcher) IndexStore() *indexstore.Store {
	return s.indexStore
}

// CoordinatorStore returns the coordinator store.
func (s *Searcher) CoordinatorStore() *CoordinatorStore {
	return s.coordinator
}

// generateID creates a unique random document ID.
func generateID() string {
	b := make([]byte, 12)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// IndexDoc proposes an atomic document write and Bleve index mutation through Raft consensus.
// Because Plexus ensures quorum commitment AND local state machine application before returning,
// this method returns only when the document is guaranteed to be in both local memory stores.
func (s *Searcher) IndexDoc(ctx context.Context, index, id string, data json.RawMessage, metadata map[string]string) (*docstore.Document, error) {
	if index == "" {
		return nil, ErrMissingIndex
	}
	if id == "" {
		id = generateID()
	}

	req := AtomicIndexRequest{
		Index:    index,
		ID:       id,
		Data:     data,
		Metadata: metadata,
	}

	s.mu.RLock()
	m := s.mutator
	s.mu.RUnlock()

	if m == nil {
		resp, err := s.coordinator.handleAtomicIndex(ctx, req)
		if err != nil {
			return nil, err
		}
		return resp.Document, nil
	}

	res, err := m.Apply(ctx, CmdAtomicIndex, req)
	if err != nil {
		return nil, err
	}

	if resp, ok := res.(*AtomicIndexResponse); ok {
		return resp.Document, nil
	}
	if resp, ok := res.(AtomicIndexResponse); ok {
		return resp.Document, nil
	}

	var resp AtomicIndexResponse
	b, err := json.Marshal(res)
	if err == nil && json.Unmarshal(b, &resp) == nil && resp.Document != nil {
		return resp.Document, nil
	}

	return nil, fmt.Errorf("unexpected index response type: %T", res)
}

// GetDoc fetches a document directly from the local node's in-memory docstore with zero network hops.
// Monotonic sequential read consistency is guaranteed by Plexus.
func (s *Searcher) GetDoc(ctx context.Context, index, id string) (*docstore.Document, error) {
	doc, found := s.docStore.Get(index, id)
	if !found || doc == nil {
		return nil, ErrNotFound
	}
	return doc, nil
}

// DeleteDoc proposes an atomic document deletion across both stores.
func (s *Searcher) DeleteDoc(ctx context.Context, index, id string) error {
	if index == "" {
		return ErrMissingIndex
	}
	if id == "" {
		return docstore.ErrMissingID
	}

	req := AtomicDeleteRequest{
		Index: index,
		ID:    id,
	}

	s.mu.RLock()
	m := s.mutator
	s.mu.RUnlock()

	if m == nil {
		_, err := s.coordinator.handleAtomicDelete(ctx, req)
		return err
	}

	res, err := m.Apply(ctx, CmdAtomicDelete, req)
	if err != nil {
		return err
	}

	if resp, ok := res.(*AtomicDeleteResponse); ok {
		if !resp.Deleted {
			return ErrNotFound
		}
		return nil
	}

	var resp AtomicDeleteResponse
	b, err := json.Marshal(res)
	if err == nil && json.Unmarshal(b, &resp) == nil {
		if !resp.Deleted {
			return ErrNotFound
		}
		return nil
	}

	return nil
}

// Batch proposes an atomic batch of indexing and deleting across both stores.
func (s *Searcher) Batch(ctx context.Context, req AtomicBatchRequest) (*AtomicBatchResponse, error) {
	if req.Index == "" {
		return nil, ErrMissingIndex
	}

	s.mu.RLock()
	m := s.mutator
	s.mu.RUnlock()

	if m == nil {
		return s.coordinator.handleAtomicBatch(ctx, req)
	}

	res, err := m.Apply(ctx, CmdAtomicBatch, req)
	if err != nil {
		return nil, err
	}

	if resp, ok := res.(*AtomicBatchResponse); ok {
		return resp, nil
	}
	if resp, ok := res.(AtomicBatchResponse); ok {
		return &resp, nil
	}

	var resp AtomicBatchResponse
	b, err := json.Marshal(res)
	if err == nil && json.Unmarshal(b, &resp) == nil {
		return &resp, nil
	}

	return nil, fmt.Errorf("unexpected batch response type: %T", res)
}

// Search executes a local in-memory Bleve full-text query with microsecond latency and zero CDC lag.
// If req.LoadDocs is true, matching hits are enriched with the full document from docstore.
func (s *Searcher) Search(ctx context.Context, index string, req indexstore.SearchRequest) (*indexstore.SearchResult, error) {
	result, err := s.indexStore.Search(index, req)
	if err != nil {
		return nil, err
	}

	if req.LoadDocs && len(result.Hits) > 0 {
		for i := range result.Hits {
			if doc, found := s.docStore.Get(index, result.Hits[i].ID); found {
				result.Hits[i].Document = doc
			}
		}
	}

	return result, nil
}

// DocCount returns the count of documents in the index.
func (s *Searcher) DocCount(index string) int {
	return s.docStore.Count(index)
}

// ListIndexes returns all active indexes.
func (s *Searcher) ListIndexes() []string {
	return s.docStore.ListIndexes()
}

// ClusterStatus compiles consensus and node health status.
func (s *Searcher) ClusterStatus() *ClusterStatus {
	s.mu.RLock()
	c := s.cluster
	s.mu.RUnlock()

	status := &ClusterStatus{
		Indexes: make(map[string]IndexStatistics),
		Nodes:   make([]ClusterNodeInfo, 0),
		Uptime:  time.Since(s.startTime).Truncate(time.Second).String(),
	}

	// Index statistics
	for _, idx := range s.docStore.ListIndexes() {
		status.Indexes[idx] = IndexStatistics{
			DocCount: s.docStore.Count(idx),
		}
	}

	if c == nil {
		status.NodeID = "standalone"
		status.IsLeader = true
		status.RaftState = "Standalone"
		return status
	}

	mach := c.DefaultRaftMachine()
	status.NodeID = c.Me().ID
	status.IsLeader = mach.IsLeader()
	status.CommittedIndex = mach.LastCommittedIndex()

	leader := mach.Leader()
	if leader != nil {
		status.LeaderAddress = leader.Address
		if leader.ID != "" {
			status.LeaderID = leader.ID
		}
	}

	if mach.IsLeader() {
		status.RaftState = "Leader"
	} else if leader != nil {
		status.RaftState = "Follower"
	} else {
		status.RaftState = "Candidate"
	}

	if members, err := mach.Members(); err == nil {
		for _, m := range members {
			isLeader := (leader != nil && leader.Address == m.Address)
			status.Nodes = append(status.Nodes, ClusterNodeInfo{
				ID:      m.ID,
				Address: m.Address,
				Voter:   m.Voter,
				Leader:  isLeader,
			})
		}
	}

	return status
}
