package api

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/avivklas/plexus-search/pkg/docstore"
	"github.com/avivklas/plexus-search/pkg/indexstore"
	"github.com/avivklas/plexus-search/pkg/searcher"
)

func (s *Server) registerRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/v1/indexes/{index}/docs", s.handleIndexDoc)
	mux.HandleFunc("GET /api/v1/indexes/{index}/docs/{id}", s.handleGetDoc)
	mux.HandleFunc("DELETE /api/v1/indexes/{index}/docs/{id}", s.handleDeleteDoc)
	mux.HandleFunc("POST /api/v1/indexes/{index}/search", s.handleSearch)
	mux.HandleFunc("GET /api/v1/cluster/status", s.handleClusterStatus)

	// Additional utility endpoints
	mux.HandleFunc("POST /api/v1/indexes/{index}/batch", s.handleBatch)
	mux.HandleFunc("GET /api/v1/indexes", s.handleListIndexes)
	mux.HandleFunc("GET /api/v1/health", s.handleHealth)
	mux.HandleFunc("GET /health", s.handleHealth)
}

// IndexDocEnvelope is used when documents are submitted in an envelope.
type IndexDocEnvelope struct {
	ID       string            `json:"id,omitempty"`
	Data     json.RawMessage   `json:"data,omitempty"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

// handleIndexDoc handles POST /api/v1/indexes/{index}/docs
func (s *Server) handleIndexDoc(w http.ResponseWriter, r *http.Request) {
	index := r.PathValue("index")
	if index == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "index is required"})
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil || len(body) == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "request body cannot be empty"})
		return
	}

	docID := r.URL.Query().Get("id")
	var docData json.RawMessage
	var metadata map[string]string

	// Try unmarshaling as envelope
	var env IndexDocEnvelope
	if err := json.Unmarshal(body, &env); err == nil && len(env.Data) > 0 {
		if env.ID != "" {
			docID = env.ID
		}
		docData = env.Data
		metadata = env.Metadata
	} else {
		// Treat entire body as the document
		var rawMap map[string]any
		if err := json.Unmarshal(body, &rawMap); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON body: " + err.Error()})
			return
		}

		if docID == "" {
			if idVal, ok := rawMap["id"].(string); ok && idVal != "" {
				docID = idVal
			} else if idVal, ok := rawMap["_id"].(string); ok && idVal != "" {
				docID = idVal
			}
		}
		docData = body
	}

	doc, err := s.searcher.IndexDoc(r.Context(), index, docID, docData, metadata)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{
		"id":         doc.ID,
		"index":      doc.Index,
		"revision":   doc.Revision,
		"created_at": doc.CreatedAt,
		"updated_at": doc.UpdatedAt,
	})
}

// handleGetDoc handles GET /api/v1/indexes/{index}/docs/{id}
func (s *Server) handleGetDoc(w http.ResponseWriter, r *http.Request) {
	index := r.PathValue("index")
	id := r.PathValue("id")

	if index == "" || id == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "index and id are required"})
		return
	}

	doc, err := s.searcher.GetDoc(r.Context(), index, id)
	if err != nil {
		if errors.Is(err, searcher.ErrNotFound) || errors.Is(err, docstore.ErrNotFound) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "document not found"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	var dataParsed any
	if len(doc.Data) > 0 {
		_ = json.Unmarshal(doc.Data, &dataParsed)
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"id":         doc.ID,
		"index":      doc.Index,
		"revision":   doc.Revision,
		"data":       dataParsed,
		"metadata":   doc.Metadata,
		"created_at": doc.CreatedAt,
		"updated_at": doc.UpdatedAt,
	})
}

// handleDeleteDoc handles DELETE /api/v1/indexes/{index}/docs/{id}
func (s *Server) handleDeleteDoc(w http.ResponseWriter, r *http.Request) {
	index := r.PathValue("index")
	id := r.PathValue("id")

	if index == "" || id == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "index and id are required"})
		return
	}

	err := s.searcher.DeleteDoc(r.Context(), index, id)
	if err != nil {
		if errors.Is(err, searcher.ErrNotFound) || errors.Is(err, docstore.ErrNotFound) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "document not found"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"index":   index,
		"id":      id,
		"deleted": true,
	})
}

// handleSearch handles POST /api/v1/indexes/{index}/search
func (s *Server) handleSearch(w http.ResponseWriter, r *http.Request) {
	index := r.PathValue("index")
	if index == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "index is required"})
		return
	}

	var searchReq indexstore.SearchRequest
	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "read request body: " + err.Error()})
		return
	}

	if len(body) > 0 {
		if err := json.Unmarshal(body, &searchReq); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid search request JSON: " + err.Error()})
			return
		}
	}

	// Always default load_docs to true if query requested it or not specified
	if r.URL.Query().Get("load_docs") == "true" {
		searchReq.LoadDocs = true
	}

	res, err := s.searcher.Search(r.Context(), index, searchReq)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "search failed: " + err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, res)
}

// handleClusterStatus handles GET /api/v1/cluster/status
func (s *Server) handleClusterStatus(w http.ResponseWriter, r *http.Request) {
	status := s.searcher.ClusterStatus()
	writeJSON(w, http.StatusOK, status)
}

// handleBatch handles POST /api/v1/indexes/{index}/batch
func (s *Server) handleBatch(w http.ResponseWriter, r *http.Request) {
	index := r.PathValue("index")
	if index == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "index is required"})
		return
	}

	var batchReq searcher.AtomicBatchRequest
	if err := json.NewDecoder(r.Body).Decode(&batchReq); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid batch JSON: " + err.Error()})
		return
	}
	batchReq.Index = index

	resp, err := s.searcher.Batch(r.Context(), batchReq)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, resp)
}

// handleListIndexes handles GET /api/v1/indexes
func (s *Server) handleListIndexes(w http.ResponseWriter, r *http.Request) {
	indexes := s.searcher.ListIndexes()
	result := make(map[string]any)
	for _, idx := range indexes {
		result[idx] = map[string]any{
			"doc_count": s.searcher.DocCount(idx),
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"indexes": result,
	})
}

// handleHealth handles GET /api/v1/health and GET /health
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"status": "ok",
		"time":   "ok",
	})
}
