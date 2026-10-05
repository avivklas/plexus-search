package docstore

import (
	"encoding/json"
	"time"

	"github.com/avivklas/plexus"
)

const (
	CmdPutDoc    plexus.CommandType = "docstore.put"
	CmdGetDoc    plexus.CommandType = "docstore.get"
	CmdDeleteDoc plexus.CommandType = "docstore.delete"
	CmdBatchDocs plexus.CommandType = "docstore.batch"
)

// PutDocRequest represents a request to insert or update a document.
type PutDocRequest struct {
	Index    string            `json:"index"`
	ID       string            `json:"id"`
	Data     json.RawMessage   `json:"data"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

// PutDocResponse contains the result of a document put operation.
type PutDocResponse struct {
	Document *Document `json:"document"`
}

// GetDocRequest represents a request to fetch a document.
type GetDocRequest struct {
	Index string `json:"index"`
	ID    string `json:"id"`
}

// GetDocResponse represents the response containing the fetched document.
type GetDocResponse struct {
	Document *Document `json:"document,omitempty"`
	Found    bool      `json:"found"`
}

// DeleteDocRequest represents a request to delete a document.
type DeleteDocRequest struct {
	Index string `json:"index"`
	ID    string `json:"id"`
}

// DeleteDocResponse represents the response from deleting a document.
type DeleteDocResponse struct {
	Index   string `json:"index"`
	ID      string `json:"id"`
	Deleted bool   `json:"deleted"`
}

// BatchDocsRequest represents an atomic batch of put and delete operations.
type BatchDocsRequest struct {
	Index   string             `json:"index"`
	Puts    []PutDocRequest    `json:"puts,omitempty"`
	Deletes []DeleteDocRequest `json:"deletes,omitempty"`
}

// BatchDocsResponse contains the counts of batch operations applied.
type BatchDocsResponse struct {
	PutCount    int `json:"put_count"`
	DeleteCount int `json:"delete_count"`
}

// SnapshotData represents the full snapshot payload for docstore.
type SnapshotData struct {
	Timestamp time.Time                       `json:"timestamp"`
	Documents map[string]map[string]*Document `json:"documents"` // index -> docID -> Document
}
