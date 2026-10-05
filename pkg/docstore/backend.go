package docstore

import (
	"fmt"
	"sync"
)

// Backend persists raw documents. The Store keeps only lightweight per-index
// counters in memory; all document bodies live in the Backend.
//
// Implementations must be safe for concurrent use.
type Backend interface {
	// Get returns the document or (nil, nil) if it does not exist.
	Get(index, id string) (*Document, error)
	// Write applies puts and deletes. Implementations should make it atomic
	// when the underlying storage supports it.
	Write(puts []*Document, deletes []DocKey) error
	// Scan calls fn for every stored document. Iteration stops on fn error.
	Scan(fn func(*Document) error) error
	// Counts returns the number of documents per index.
	Counts() (map[string]int, error)
	// Reset removes all documents.
	Reset() error
	// Close releases backend resources.
	Close() error
}

// DocKey identifies a document.
type DocKey struct {
	Index string
	ID    string
}

// BackendType names a supported storage backend.
type BackendType string

const (
	BackendMem    BackendType = "mem"
	BackendPebble BackendType = "pebble"
	BackendS3     BackendType = "s3"
)

// Config selects and configures the document storage backend.
type Config struct {
	Type   BackendType
	Pebble PebbleConfig
	S3     S3Config
}

// OpenBackend creates a Backend from cfg. An empty Type means in-memory.
func OpenBackend(cfg Config) (Backend, error) {
	switch cfg.Type {
	case "", BackendMem:
		return NewMemBackend(), nil
	case BackendPebble:
		return NewPebbleBackend(cfg.Pebble)
	case BackendS3:
		return NewS3Backend(cfg.S3)
	default:
		return nil, fmt.Errorf("unknown docstore backend %q (want mem, pebble or s3)", cfg.Type)
	}
}

// MemBackend keeps documents in process memory.
type MemBackend struct {
	mu   sync.RWMutex
	docs map[string]map[string]*Document
}

// NewMemBackend returns an empty in-memory backend.
func NewMemBackend() *MemBackend {
	return &MemBackend{docs: make(map[string]map[string]*Document)}
}

func (m *MemBackend) Get(index, id string) (*Document, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.docs[index][id].Clone(), nil
}

func (m *MemBackend) Write(puts []*Document, deletes []DocKey) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, d := range puts {
		idx, ok := m.docs[d.Index]
		if !ok {
			idx = make(map[string]*Document)
			m.docs[d.Index] = idx
		}
		idx[d.ID] = d.Clone()
	}
	for _, k := range deletes {
		delete(m.docs[k.Index], k.ID)
	}
	return nil
}

func (m *MemBackend) Scan(fn func(*Document) error) error {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, idx := range m.docs {
		for _, d := range idx {
			if err := fn(d.Clone()); err != nil {
				return err
			}
		}
	}
	return nil
}

func (m *MemBackend) Counts() (map[string]int, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make(map[string]int, len(m.docs))
	for idx, docs := range m.docs {
		if len(docs) > 0 {
			out[idx] = len(docs)
		}
	}
	return out, nil
}

func (m *MemBackend) Reset() error {
	m.mu.Lock()
	m.docs = make(map[string]map[string]*Document)
	m.mu.Unlock()
	return nil
}

func (m *MemBackend) Close() error { return nil }
