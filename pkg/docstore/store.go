package docstore

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/avivklas/plexus"
)

var (
	ErrMissingIndex = errors.New("missing index name")
	ErrMissingID    = errors.New("missing document id")
	ErrNotFound     = errors.New("document not found")
)

// Store implements plexus.Store to store and replicate raw JSON documents.
type Store struct {
	plexus.BaseStore
	mu      sync.RWMutex
	docs    map[string]map[string]*Document // index -> docID -> Document
	mutator plexus.Mutator
}

// New creates an initialized document store.
func New() *Store {
	s := &Store{
		BaseStore: plexus.NewBaseStore(),
		docs:      make(map[string]map[string]*Document),
	}

	// Register Raft mutating and state machine handlers
	s.Handle(CmdPutDoc, s.handlePut)
	s.Handle(CmdGetDoc, s.handleGet)
	s.Handle(CmdDeleteDoc, s.handleDelete)
	s.Handle(CmdBatchDocs, s.handleBatch)

	return s
}

// ID implements plexus.Store.
func (s *Store) ID() plexus.StoreID {
	return "docstore"
}

// AttachMutator attaches the Plexus cluster mutator for proposals.
func (s *Store) AttachMutator(m plexus.Mutator) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.mutator = m
}

func (s *Store) handlePut(ctx context.Context, req PutDocRequest) (*PutDocResponse, error) {
	doc, err := s.ApplyPut(ctx, req)
	if err != nil {
		return nil, err
	}
	return &PutDocResponse{Document: doc}, nil
}

func (s *Store) handleGet(ctx context.Context, req GetDocRequest) (*GetDocResponse, error) {
	doc, found := s.Get(req.Index, req.ID)
	return &GetDocResponse{Document: doc, Found: found}, nil
}

func (s *Store) handleDelete(ctx context.Context, req DeleteDocRequest) (*DeleteDocResponse, error) {
	deleted, err := s.ApplyDelete(ctx, req)
	if err != nil {
		return nil, err
	}
	return &DeleteDocResponse{
		Index:   req.Index,
		ID:      req.ID,
		Deleted: deleted,
	}, nil
}

func (s *Store) handleBatch(ctx context.Context, req BatchDocsRequest) (*BatchDocsResponse, error) {
	return s.ApplyBatch(ctx, req)
}

// ApplyPut applies a document insertion or update directly to the local state machine.
// This is called during FSM apply or by high-level atomic coordinators.
func (s *Store) ApplyPut(ctx context.Context, req PutDocRequest) (*Document, error) {
	if req.Index == "" {
		return nil, ErrMissingIndex
	}
	if req.ID == "" {
		return nil, ErrMissingID
	}

	now := time.Now().UTC()

	s.mu.Lock()
	defer s.mu.Unlock()

	idxDocs, ok := s.docs[req.Index]
	if !ok {
		idxDocs = make(map[string]*Document)
		s.docs[req.Index] = idxDocs
	}

	existing, exists := idxDocs[req.ID]
	var rev int64 = 1
	createdAt := now

	if exists {
		rev = existing.Revision + 1
		createdAt = existing.CreatedAt
	}

	doc := &Document{
		ID:        req.ID,
		Index:     req.Index,
		Revision:  rev,
		Data:      req.Data,
		Metadata:  req.Metadata,
		CreatedAt: createdAt,
		UpdatedAt: now,
	}

	idxDocs[req.ID] = doc
	return doc.Clone(), nil
}

// ApplyDelete removes a document directly from the local state machine.
func (s *Store) ApplyDelete(ctx context.Context, req DeleteDocRequest) (bool, error) {
	if req.Index == "" {
		return false, ErrMissingIndex
	}
	if req.ID == "" {
		return false, ErrMissingID
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	idxDocs, ok := s.docs[req.Index]
	if !ok {
		return false, nil
	}

	_, exists := idxDocs[req.ID]
	if !exists {
		return false, nil
	}

	delete(idxDocs, req.ID)
	return true, nil
}

// ApplyBatch applies a batch of put and delete operations atomically to the local state machine.
func (s *Store) ApplyBatch(ctx context.Context, req BatchDocsRequest) (*BatchDocsResponse, error) {
	if req.Index == "" {
		return nil, ErrMissingIndex
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	idxDocs, ok := s.docs[req.Index]
	if !ok {
		idxDocs = make(map[string]*Document)
		s.docs[req.Index] = idxDocs
	}

	now := time.Now().UTC()
	var putCount, delCount int

	for _, p := range req.Puts {
		if p.ID == "" {
			continue
		}
		existing, exists := idxDocs[p.ID]
		var rev int64 = 1
		createdAt := now
		if exists {
			rev = existing.Revision + 1
			createdAt = existing.CreatedAt
		}

		idxDocs[p.ID] = &Document{
			ID:        p.ID,
			Index:     req.Index,
			Revision:  rev,
			Data:      p.Data,
			Metadata:  p.Metadata,
			CreatedAt: createdAt,
			UpdatedAt: now,
		}
		putCount++
	}

	for _, d := range req.Deletes {
		if d.ID == "" {
			continue
		}
		if _, exists := idxDocs[d.ID]; exists {
			delete(idxDocs, d.ID)
			delCount++
		}
	}

	return &BatchDocsResponse{
		PutCount:    putCount,
		DeleteCount: delCount,
	}, nil
}

// Put proposes a document put through the Plexus cluster consensus.
// If no mutator is attached (e.g. standalone local mode), it applies directly.
func (s *Store) Put(ctx context.Context, req PutDocRequest) (*Document, error) {
	s.mu.RLock()
	m := s.mutator
	s.mu.RUnlock()

	if m == nil {
		return s.ApplyPut(ctx, req)
	}

	res, err := m.Apply(ctx, CmdPutDoc, req)
	if err != nil {
		return nil, err
	}

	if resp, ok := res.(*PutDocResponse); ok {
		return resp.Document, nil
	}
	if resp, ok := res.(PutDocResponse); ok {
		return resp.Document, nil
	}

	// In case of generic unmarshaling
	var resp PutDocResponse
	b, err := json.Marshal(res)
	if err == nil && json.Unmarshal(b, &resp) == nil && resp.Document != nil {
		return resp.Document, nil
	}

	return nil, fmt.Errorf("unexpected put response type: %T", res)
}

// Get executes a zero-hop in-memory read from the local node.
// Under Plexus's sequential consistency guarantee, this is always safe and strongly consistent
// with previously acknowledged writes on this node.
func (s *Store) Get(index, id string) (*Document, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	idxDocs, ok := s.docs[index]
	if !ok {
		return nil, false
	}

	doc, ok := idxDocs[id]
	if !ok {
		return nil, false
	}

	return doc.Clone(), true
}

// Delete proposes a document deletion through the Plexus cluster.
func (s *Store) Delete(ctx context.Context, req DeleteDocRequest) (bool, error) {
	s.mu.RLock()
	m := s.mutator
	s.mu.RUnlock()

	if m == nil {
		return s.ApplyDelete(ctx, req)
	}

	res, err := m.Apply(ctx, CmdDeleteDoc, req)
	if err != nil {
		return false, err
	}

	if resp, ok := res.(*DeleteDocResponse); ok {
		return resp.Deleted, nil
	}
	if resp, ok := res.(DeleteDocResponse); ok {
		return resp.Deleted, nil
	}

	var resp DeleteDocResponse
	b, err := json.Marshal(res)
	if err == nil && json.Unmarshal(b, &resp) == nil {
		return resp.Deleted, nil
	}

	return false, fmt.Errorf("unexpected delete response type: %T", res)
}

// Batch proposes a batch mutation through the Plexus cluster.
func (s *Store) Batch(ctx context.Context, req BatchDocsRequest) (*BatchDocsResponse, error) {
	s.mu.RLock()
	m := s.mutator
	s.mu.RUnlock()

	if m == nil {
		return s.ApplyBatch(ctx, req)
	}

	res, err := m.Apply(ctx, CmdBatchDocs, req)
	if err != nil {
		return nil, err
	}

	if resp, ok := res.(*BatchDocsResponse); ok {
		return resp, nil
	}
	if resp, ok := res.(BatchDocsResponse); ok {
		return &resp, nil
	}

	var resp BatchDocsResponse
	b, err := json.Marshal(res)
	if err == nil && json.Unmarshal(b, &resp) == nil {
		return &resp, nil
	}

	return nil, fmt.Errorf("unexpected batch response type: %T", res)
}

// Count returns the number of documents in an index.
func (s *Store) Count(index string) int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	idxDocs, ok := s.docs[index]
	if !ok {
		return 0
	}
	return len(idxDocs)
}

// ListIndexes returns all known index names.
func (s *Store) ListIndexes() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	indexes := make([]string, 0, len(s.docs))
	for idx := range s.docs {
		indexes = append(indexes, idx)
	}
	return indexes
}

// AllDocs returns clones of all documents in the specified index.
func (s *Store) AllDocs(index string) []*Document {
	s.mu.RLock()
	defer s.mu.RUnlock()

	idxDocs, ok := s.docs[index]
	if !ok {
		return nil
	}

	result := make([]*Document, 0, len(idxDocs))
	for _, doc := range idxDocs {
		result = append(result, doc.Clone())
	}
	return result
}

// Snapshot serializes all documents into an immutable snapshot payload.
func (s *Store) Snapshot() ([]byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	snapDocs := make(map[string]map[string]*Document, len(s.docs))
	for idx, docsMap := range s.docs {
		copiedMap := make(map[string]*Document, len(docsMap))
		for id, doc := range docsMap {
			copiedMap[id] = doc.Clone()
		}
		snapDocs[idx] = copiedMap
	}

	data := SnapshotData{
		Timestamp: time.Now().UTC(),
		Documents: snapDocs,
	}

	return json.Marshal(data)
}

// Restore resets the store state from a snapshot byte slice.
func (s *Store) Restore(data []byte) error {
	var snap SnapshotData
	if err := json.Unmarshal(data, &snap); err != nil {
		return fmt.Errorf("unmarshal docstore snapshot: %w", err)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.docs = make(map[string]map[string]*Document, len(snap.Documents))
	for idx, docsMap := range snap.Documents {
		s.docs[idx] = make(map[string]*Document, len(docsMap))
		for id, doc := range docsMap {
			s.docs[idx][id] = doc.Clone()
		}
	}

	return nil
}
