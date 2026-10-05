package docstore

import (
	"encoding/json"
	"time"
)

// Document represents a raw JSON document with metadata, revision, and timestamps.
type Document struct {
	ID        string            `json:"id"`
	Index     string            `json:"index"`
	Revision  int64             `json:"revision"`
	Data      json.RawMessage   `json:"data"`
	Metadata  map[string]string `json:"metadata,omitempty"`
	CreatedAt time.Time         `json:"created_at"`
	UpdatedAt time.Time         `json:"updated_at"`
}

// Clone returns a deep copy of the document.
func (d *Document) Clone() *Document {
	if d == nil {
		return nil
	}
	dataCopy := make(json.RawMessage, len(d.Data))
	copy(dataCopy, d.Data)

	var metaCopy map[string]string
	if d.Metadata != nil {
		metaCopy = make(map[string]string, len(d.Metadata))
		for k, v := range d.Metadata {
			metaCopy[k] = v
		}
	}

	return &Document{
		ID:        d.ID,
		Index:     d.Index,
		Revision:  d.Revision,
		Data:      dataCopy,
		Metadata:  metaCopy,
		CreatedAt: d.CreatedAt,
		UpdatedAt: d.UpdatedAt,
	}
}
