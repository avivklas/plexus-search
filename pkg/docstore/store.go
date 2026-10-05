package docstore

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
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
// Document bodies live in a pluggable Backend (memory, Pebble or S3); only
// per-index document counts are held in memory.
type Store struct {
	plexus.BaseStore
	mu      sync.RWMutex
	backend Backend
	counts  map[string]int // index -> number of docs
	mutator plexus.Mutator
}

// New creates a document store backed by process memory.
func New() *Store {
	s, _ := NewWithBackend(NewMemBackend())
	return s
}

// Open creates a document store using the backend selected by cfg.
func Open(cfg Config) (*Store, error) {
	b, err := OpenBackend(cfg)
	if err != nil {
		return nil, err
	}
	s, err := NewWithBackend(b)
	if err != nil {
		b.Close()
		return nil, err
	}
	return s, nil
}

// NewWithBackend creates a document store on top of an existing backend,
// loading the per-index counters from it.
func NewWithBackend(b Backend) (*Store, error) {
	counts, err := b.Counts()
	if err != nil {
		return nil, fmt.Errorf("load docstore counts: %w", err)
	}
	s := &Store{
		BaseStore: plexus.NewBaseStore(),
		backend:   b,
		counts:    counts,
	}

	// Register Raft mutating and state machine handlers
	s.Handle(CmdPutDoc, s.handlePut)
	s.Handle(CmdGetDoc, s.handleGet)
	s.Handle(CmdDeleteDoc, s.handleDelete)
	s.Handle(CmdBatchDocs, s.handleBatch)

	return s, nil
}

// Close releases the underlying backend.
func (s *Store) Close() error {
	return s.backend.Close()
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

	existing, err := s.backend.Get(req.Index, req.ID)
	if err != nil {
		return nil, err
	}

	var rev int64 = 1
	createdAt := now
	if existing != nil {
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

	if err := s.backend.Write([]*Document{doc}, nil); err != nil {
		return nil, err
	}
	if existing == nil {
		s.counts[req.Index]++
	}
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

	existing, err := s.backend.Get(req.Index, req.ID)
	if err != nil {
		return false, err
	}
	if existing == nil {
		return false, nil
	}

	if err := s.backend.Write(nil, []DocKey{{req.Index, req.ID}}); err != nil {
		return false, err
	}
	s.decr(req.Index, 1)
	return true, nil
}

// ApplyBatch applies a batch of put and delete operations atomically to the local state machine.
func (s *Store) ApplyBatch(ctx context.Context, req BatchDocsRequest) (*BatchDocsResponse, error) {
	if req.Index == "" {
		return nil, ErrMissingIndex
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UTC()
	var putCount, delCount, delta int

	// pending holds the in-batch view of touched docs (nil value = deleted),
	// so repeated ids within a batch see each other's effects.
	pending := make(map[string]*Document)
	lookup := func(id string) (*Document, error) {
		if d, ok := pending[id]; ok {
			return d, nil
		}
		return s.backend.Get(req.Index, id)
	}

	for _, p := range req.Puts {
		if p.ID == "" {
			continue
		}
		existing, err := lookup(p.ID)
		if err != nil {
			return nil, err
		}
		var rev int64 = 1
		createdAt := now
		if existing != nil {
			rev = existing.Revision + 1
			createdAt = existing.CreatedAt
		} else {
			delta++
		}
		pending[p.ID] = &Document{
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
		existing, err := lookup(d.ID)
		if err != nil {
			return nil, err
		}
		if existing != nil {
			pending[d.ID] = nil
			delta--
			delCount++
		}
	}

	var puts []*Document
	var dels []DocKey
	for id, d := range pending {
		if d != nil {
			puts = append(puts, d)
		} else {
			dels = append(dels, DocKey{req.Index, id})
		}
	}
	if err := s.backend.Write(puts, dels); err != nil {
		return nil, err
	}
	if delta >= 0 {
		s.counts[req.Index] += delta
	} else {
		s.decr(req.Index, -delta)
	}

	return &BatchDocsResponse{
		PutCount:    putCount,
		DeleteCount: delCount,
	}, nil
}

// decr lowers an index counter, dropping it at zero. Caller holds s.mu.
func (s *Store) decr(index string, n int) {
	s.counts[index] -= n
	if s.counts[index] <= 0 {
		delete(s.counts, index)
	}
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

// Get reads a document from the local node's backend.
// Under Plexus's sequential consistency guarantee, this is always safe and strongly consistent
// with previously acknowledged writes on this node. Backend read errors are logged
// and reported as not found.
func (s *Store) Get(index, id string) (*Document, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	doc, err := s.backend.Get(index, id)
	if err != nil {
		log.Printf("docstore: get %s/%s: %v", index, id, err)
		return nil, false
	}
	if doc == nil {
		return nil, false
	}
	return doc, true
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
	return s.counts[index]
}

// ListIndexes returns all known index names.
func (s *Store) ListIndexes() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	indexes := make([]string, 0, len(s.counts))
	for idx := range s.counts {
		indexes = append(indexes, idx)
	}
	return indexes
}

// AllDocs returns all documents in the specified index.
func (s *Store) AllDocs(index string) []*Document {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []*Document
	err := s.backend.Scan(func(d *Document) error {
		if d.Index == index {
			result = append(result, d)
		}
		return nil
	})
	if err != nil {
		log.Printf("docstore: scan %s: %v", index, err)
		return nil
	}
	return result
}

// Snapshot serializes all documents into an immutable snapshot payload.
func (s *Store) Snapshot() ([]byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	snapDocs := make(map[string]map[string]*Document, len(s.counts))
	err := s.backend.Scan(func(d *Document) error {
		m, ok := snapDocs[d.Index]
		if !ok {
			m = make(map[string]*Document)
			snapDocs[d.Index] = m
		}
		m[d.ID] = d
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("scan docstore: %w", err)
	}

	data := SnapshotData{
		Timestamp: time.Now().UTC(),
		Documents: snapDocs,
	}

	return json.Marshal(data)
}

const restoreChunk = 1000

// Restore resets the store state from a snapshot byte slice.
func (s *Store) Restore(data []byte) error {
	var snap SnapshotData
	if err := json.Unmarshal(data, &snap); err != nil {
		return fmt.Errorf("unmarshal docstore snapshot: %w", err)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.backend.Reset(); err != nil {
		return fmt.Errorf("reset docstore backend: %w", err)
	}

	counts := make(map[string]int, len(snap.Documents))
	chunk := make([]*Document, 0, restoreChunk)
	flush := func() error {
		if len(chunk) == 0 {
			return nil
		}
		err := s.backend.Write(chunk, nil)
		chunk = chunk[:0]
		return err
	}
	for idx, docsMap := range snap.Documents {
		for id, doc := range docsMap {
			doc.Index, doc.ID = idx, id
			chunk = append(chunk, doc)
			counts[idx]++
			if len(chunk) == restoreChunk {
				if err := flush(); err != nil {
					return fmt.Errorf("restore docstore: %w", err)
				}
			}
		}
	}
	if err := flush(); err != nil {
		return fmt.Errorf("restore docstore: %w", err)
	}

	s.counts = counts
	return nil
}
