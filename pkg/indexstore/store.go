package indexstore

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/avivklas/plexus"
	"github.com/blevesearch/bleve/v2"
)

var (
	ErrMissingIndex = errors.New("missing index name")
	ErrMissingID    = errors.New("missing document id")
	ErrIndexNotFound = errors.New("index not found")
)

// Store implements plexus.Store wrapping in-memory Bleve inverted indexes.
type Store struct {
	plexus.BaseStore
	mu      sync.RWMutex
	indexes map[string]bleve.Index
	docs    map[string]map[string]json.RawMessage // index -> docID -> rawJSON
	mutator plexus.Mutator
}

// New creates an initialized in-memory Bleve index store.
func New() *Store {
	s := &Store{
		BaseStore: plexus.NewBaseStore(),
		indexes:   make(map[string]bleve.Index),
		docs:      make(map[string]map[string]json.RawMessage),
	}

	// Register Raft mutating handlers
	s.Handle(CmdIndexDoc, s.handleIndex)
	s.Handle(CmdDeleteDocIndex, s.handleDelete)
	s.Handle(CmdBatchIndex, s.handleBatch)

	return s
}

// ID implements plexus.Store.
func (s *Store) ID() plexus.StoreID {
	return "indexstore"
}

// AttachMutator attaches the Plexus cluster mutator.
func (s *Store) AttachMutator(m plexus.Mutator) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.mutator = m
}

func (s *Store) handleIndex(ctx context.Context, req IndexDocRequest) (*IndexDocResponse, error) {
	if err := s.ApplyIndex(ctx, req); err != nil {
		return nil, err
	}
	return &IndexDocResponse{
		Index:   req.Index,
		ID:      req.ID,
		Success: true,
	}, nil
}

func (s *Store) handleDelete(ctx context.Context, req DeleteDocIndexRequest) (*DeleteDocIndexResponse, error) {
	if err := s.ApplyDelete(ctx, req); err != nil {
		return nil, err
	}
	return &DeleteDocIndexResponse{
		Index:   req.Index,
		ID:      req.ID,
		Deleted: true,
	}, nil
}

func (s *Store) handleBatch(ctx context.Context, req BatchIndexRequest) (*BatchIndexResponse, error) {
	return s.ApplyBatch(ctx, req)
}

// getOrCreateIndex retrieves an existing Bleve index or initializes a new in-memory index.
// Assumes s.mu write lock is already held.
func (s *Store) getOrCreateIndexLocked(name string) (bleve.Index, error) {
	if idx, exists := s.indexes[name]; exists {
		return idx, nil
	}

	mapping := bleve.NewIndexMapping()
	idx, err := bleve.NewMemOnly(mapping)
	if err != nil {
		return nil, fmt.Errorf("create in-memory bleve index %q: %w", name, err)
	}

	s.indexes[name] = idx
	if _, ok := s.docs[name]; !ok {
		s.docs[name] = make(map[string]json.RawMessage)
	}
	return idx, nil
}

// ApplyIndex inserts or updates a document into the in-memory Bleve inverted index.
func (s *Store) ApplyIndex(ctx context.Context, req IndexDocRequest) error {
	if req.Index == "" {
		return ErrMissingIndex
	}
	if req.ID == "" {
		return ErrMissingID
	}

	var docMap map[string]any
	if len(req.Data) > 0 {
		if err := json.Unmarshal(req.Data, &docMap); err != nil {
			return fmt.Errorf("unmarshal doc for indexing: %w", err)
		}
	} else {
		docMap = make(map[string]any)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	idx, err := s.getOrCreateIndexLocked(req.Index)
	if err != nil {
		return err
	}

	if err := idx.Index(req.ID, docMap); err != nil {
		return fmt.Errorf("bleve index: %w", err)
	}

	s.docs[req.Index][req.ID] = req.Data
	return nil
}

// ApplyDelete removes a document from the Bleve index.
func (s *Store) ApplyDelete(ctx context.Context, req DeleteDocIndexRequest) error {
	if req.Index == "" {
		return ErrMissingIndex
	}
	if req.ID == "" {
		return ErrMissingID
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	idx, exists := s.indexes[req.Index]
	if !exists {
		return nil
	}

	if err := idx.Delete(req.ID); err != nil {
		return fmt.Errorf("bleve delete: %w", err)
	}

	if docsMap, ok := s.docs[req.Index]; ok {
		delete(docsMap, req.ID)
	}
	return nil
}

// ApplyBatch executes multiple index and delete operations on Bleve using a batch.
func (s *Store) ApplyBatch(ctx context.Context, req BatchIndexRequest) (*BatchIndexResponse, error) {
	if req.Index == "" {
		return nil, ErrMissingIndex
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	idx, err := s.getOrCreateIndexLocked(req.Index)
	if err != nil {
		return nil, err
	}

	batch := idx.NewBatch()
	docsMap := s.docs[req.Index]

	var indexedCount, deletedCount int

	for _, item := range req.Indexes {
		if item.ID == "" {
			continue
		}
		var docMap map[string]any
		if len(item.Data) > 0 {
			if err := json.Unmarshal(item.Data, &docMap); err != nil {
				return nil, fmt.Errorf("unmarshal batch item %s: %w", item.ID, err)
			}
		} else {
			docMap = make(map[string]any)
		}

		if err := batch.Index(item.ID, docMap); err != nil {
			return nil, fmt.Errorf("batch index %s: %w", item.ID, err)
		}
		docsMap[item.ID] = item.Data
		indexedCount++
	}

	for _, del := range req.Deletes {
		if del.ID == "" {
			continue
		}
		batch.Delete(del.ID)
		delete(docsMap, del.ID)
		deletedCount++
	}

	if err := idx.Batch(batch); err != nil {
		return nil, fmt.Errorf("apply bleve batch: %w", err)
	}

	return &BatchIndexResponse{
		Index:        req.Index,
		IndexedCount: indexedCount,
		DeletedCount: deletedCount,
	}, nil
}

// Index proposes a document index operation through the cluster.
func (s *Store) Index(ctx context.Context, req IndexDocRequest) (*IndexDocResponse, error) {
	s.mu.RLock()
	m := s.mutator
	s.mu.RUnlock()

	if m == nil {
		if err := s.ApplyIndex(ctx, req); err != nil {
			return nil, err
		}
		return &IndexDocResponse{Index: req.Index, ID: req.ID, Success: true}, nil
	}

	res, err := m.Apply(ctx, CmdIndexDoc, req)
	if err != nil {
		return nil, err
	}

	if resp, ok := res.(*IndexDocResponse); ok {
		return resp, nil
	}
	if resp, ok := res.(IndexDocResponse); ok {
		return &resp, nil
	}

	var resp IndexDocResponse
	b, err := json.Marshal(res)
	if err == nil && json.Unmarshal(b, &resp) == nil {
		return &resp, nil
	}

	return nil, fmt.Errorf("unexpected index response type: %T", res)
}

// Delete proposes a document index deletion through the cluster.
func (s *Store) Delete(ctx context.Context, req DeleteDocIndexRequest) (*DeleteDocIndexResponse, error) {
	s.mu.RLock()
	m := s.mutator
	s.mu.RUnlock()

	if m == nil {
		if err := s.ApplyDelete(ctx, req); err != nil {
			return nil, err
		}
		return &DeleteDocIndexResponse{Index: req.Index, ID: req.ID, Deleted: true}, nil
	}

	res, err := m.Apply(ctx, CmdDeleteDocIndex, req)
	if err != nil {
		return nil, err
	}

	if resp, ok := res.(*DeleteDocIndexResponse); ok {
		return resp, nil
	}
	if resp, ok := res.(DeleteDocIndexResponse); ok {
		return &resp, nil
	}

	var resp DeleteDocIndexResponse
	b, err := json.Marshal(res)
	if err == nil && json.Unmarshal(b, &resp) == nil {
		return &resp, nil
	}

	return nil, fmt.Errorf("unexpected delete response type: %T", res)
}

// Batch proposes a batch indexing operation through the cluster.
func (s *Store) Batch(ctx context.Context, req BatchIndexRequest) (*BatchIndexResponse, error) {
	s.mu.RLock()
	m := s.mutator
	s.mu.RUnlock()

	if m == nil {
		return s.ApplyBatch(ctx, req)
	}

	res, err := m.Apply(ctx, CmdBatchIndex, req)
	if err != nil {
		return nil, err
	}

	if resp, ok := res.(*BatchIndexResponse); ok {
		return resp, nil
	}
	if resp, ok := res.(BatchIndexResponse); ok {
		return &resp, nil
	}

	var resp BatchIndexResponse
	b, err := json.Marshal(res)
	if err == nil && json.Unmarshal(b, &resp) == nil {
		return &resp, nil
	}

	return nil, fmt.Errorf("unexpected batch response type: %T", res)
}

// Search executes a local in-memory search against the named index with microsecond latency.
func (s *Store) Search(indexName string, req SearchRequest) (*SearchResult, error) {
	s.mu.RLock()
	idx, exists := s.indexes[indexName]
	s.mu.RUnlock()

	if !exists {
		return &SearchResult{Hits: []SearchHit{}}, nil
	}

	bleveReq, err := req.BuildBleveSearchRequest()
	if err != nil {
		return nil, fmt.Errorf("build search request: %w", err)
	}

	res, err := idx.Search(bleveReq)
	if err != nil {
		return nil, fmt.Errorf("bleve search: %w", err)
	}

	return ConvertBleveResult(indexName, res), nil
}

// SearchBleve executes a raw Bleve search request locally.
func (s *Store) SearchBleve(indexName string, req *bleve.SearchRequest) (*bleve.SearchResult, error) {
	s.mu.RLock()
	idx, exists := s.indexes[indexName]
	s.mu.RUnlock()

	if !exists {
		return nil, ErrIndexNotFound
	}

	return idx.Search(req)
}

// SearchMatch is a convenience method for match queries.
func (s *Store) SearchMatch(indexName, field, match string, from, size int) (*SearchResult, error) {
	return s.Search(indexName, SearchRequest{
		Query: QueryDefinition{Type: QueryTypeMatch, Field: field, Value: match},
		From:  from,
		Size:  size,
	})
}

// SearchMatchPhrase is a convenience method for match phrase queries.
func (s *Store) SearchMatchPhrase(indexName, field, phrase string, from, size int) (*SearchResult, error) {
	return s.Search(indexName, SearchRequest{
		Query: QueryDefinition{Type: QueryTypeMatchPhrase, Field: field, Value: phrase},
		From:  from,
		Size:  size,
	})
}

// SearchTerm is a convenience method for exact term queries.
func (s *Store) SearchTerm(indexName, field, term string, from, size int) (*SearchResult, error) {
	return s.Search(indexName, SearchRequest{
		Query: QueryDefinition{Type: QueryTypeTerm, Field: field, Value: term},
		From:  from,
		Size:  size,
	})
}

// SearchPrefix is a convenience method for prefix queries.
func (s *Store) SearchPrefix(indexName, field, prefix string, from, size int) (*SearchResult, error) {
	return s.Search(indexName, SearchRequest{
		Query: QueryDefinition{Type: QueryTypePrefix, Field: field, Value: prefix},
		From:  from,
		Size:  size,
	})
}

// SearchFuzzy is a convenience method for fuzzy queries.
func (s *Store) SearchFuzzy(indexName, field, term string, fuzziness, from, size int) (*SearchResult, error) {
	return s.Search(indexName, SearchRequest{
		Query: QueryDefinition{Type: QueryTypeFuzzy, Field: field, Value: term, Fuzziness: fuzziness},
		From:  from,
		Size:  size,
	})
}

// SearchNumericRange is a convenience method for numeric range queries.
func (s *Store) SearchNumericRange(indexName, field string, min, max *float64, minInc, maxInc *bool, from, size int) (*SearchResult, error) {
	return s.Search(indexName, SearchRequest{
		Query: QueryDefinition{
			Type:         QueryTypeNumericRange,
			Field:        field,
			Min:          min,
			Max:          max,
			MinInclusive: minInc,
			MaxInclusive: maxInc,
		},
		From: from,
		Size: size,
	})
}

// SearchBoolean is a convenience method for boolean queries.
func (s *Store) SearchBoolean(indexName string, must, should, mustNot []QueryDefinition, from, size int) (*SearchResult, error) {
	return s.Search(indexName, SearchRequest{
		Query: QueryDefinition{
			Type:    QueryTypeBoolean,
			Must:    must,
			Should:  should,
			MustNot: mustNot,
		},
		From: from,
		Size: size,
	})
}

// DocCount returns the number of indexed documents in the named index.
func (s *Store) DocCount(indexName string) (uint64, error) {
	s.mu.RLock()
	idx, exists := s.indexes[indexName]
	s.mu.RUnlock()

	if !exists {
		return 0, nil
	}
	return idx.DocCount()
}

// Snapshot serializes all indexed document data for replication or persistence.
func (s *Store) Snapshot() ([]byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	snapDocs := make(map[string]map[string]json.RawMessage, len(s.docs))
	for idx, docMap := range s.docs {
		copiedMap := make(map[string]json.RawMessage, len(docMap))
		for id, raw := range docMap {
			rawCopy := make(json.RawMessage, len(raw))
			copy(rawCopy, raw)
			copiedMap[id] = rawCopy
		}
		snapDocs[idx] = copiedMap
	}

	data := SnapshotData{
		Timestamp: time.Now().UTC(),
		Documents: snapDocs,
	}

	return json.Marshal(data)
}

// Restore resets the index store state and rebuilds all Bleve in-memory indexes.
func (s *Store) Restore(data []byte) error {
	var snap SnapshotData
	if err := json.Unmarshal(data, &snap); err != nil {
		return fmt.Errorf("unmarshal indexstore snapshot: %w", err)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Close existing indexes
	for _, idx := range s.indexes {
		_ = idx.Close()
	}

	s.indexes = make(map[string]bleve.Index)
	s.docs = make(map[string]map[string]json.RawMessage)

	// Rebuild in-memory indexes
	for idxName, docsMap := range snap.Documents {
		mapping := bleve.NewIndexMapping()
		idx, err := bleve.NewMemOnly(mapping)
		if err != nil {
			return fmt.Errorf("recreate index %s: %w", idxName, err)
		}
		s.indexes[idxName] = idx
		s.docs[idxName] = make(map[string]json.RawMessage)

		batch := idx.NewBatch()
		for id, raw := range docsMap {
			var docMap map[string]any
			if len(raw) > 0 {
				_ = json.Unmarshal(raw, &docMap)
			} else {
				docMap = make(map[string]any)
			}
			_ = batch.Index(id, docMap)
			s.docs[idxName][id] = raw
		}
		if err := idx.Batch(batch); err != nil {
			return fmt.Errorf("restore batch index %s: %w", idxName, err)
		}
	}

	return nil
}

// Close closes all underlying Bleve index instances.
func (s *Store) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	var firstErr error
	for _, idx := range s.indexes {
		if err := idx.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	s.indexes = make(map[string]bleve.Index)
	return firstErr
}
