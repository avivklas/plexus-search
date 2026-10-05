package indexstore

import (
	"encoding/json"
	"time"

	"github.com/avivklas/plexus"
)

const (
	CmdIndexDoc       plexus.CommandType = "indexstore.index"
	CmdDeleteDocIndex plexus.CommandType = "indexstore.delete"
	CmdBatchIndex     plexus.CommandType = "indexstore.batch"
)

// IndexDocRequest represents a request to index a document.
type IndexDocRequest struct {
	Index string          `json:"index"`
	ID    string          `json:"id"`
	Data  json.RawMessage `json:"data"`
}

// IndexDocResponse represents the response from indexing a document.
type IndexDocResponse struct {
	Index   string `json:"index"`
	ID      string `json:"id"`
	Success bool   `json:"success"`
}

// DeleteDocIndexRequest represents a request to remove a document from an index.
type DeleteDocIndexRequest struct {
	Index string `json:"index"`
	ID    string `json:"id"`
}

// DeleteDocIndexResponse represents the response from removing a document from an index.
type DeleteDocIndexResponse struct {
	Index   string `json:"index"`
	ID      string `json:"id"`
	Deleted bool   `json:"deleted"`
}

// BatchIndexRequest represents a batch of indexing and deletion operations.
type BatchIndexRequest struct {
	Index   string                  `json:"index"`
	Indexes []IndexDocRequest       `json:"indexes,omitempty"`
	Deletes []DeleteDocIndexRequest `json:"deletes,omitempty"`
}

// BatchIndexResponse contains the count of items indexed and deleted in a batch.
type BatchIndexResponse struct {
	Index        string `json:"index"`
	IndexedCount int    `json:"indexed_count"`
	DeletedCount int    `json:"deleted_count"`
}

// SnapshotData represents the snapshot format for the index store.
type SnapshotData struct {
	Timestamp time.Time                          `json:"timestamp"`
	Documents map[string]map[string]json.RawMessage `json:"documents"` // index -> docID -> rawJSON
}
