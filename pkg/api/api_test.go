package api_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/avivklas/plexus-search/pkg/api"
	"github.com/avivklas/plexus-search/pkg/docstore"
	"github.com/avivklas/plexus-search/pkg/indexstore"
	"github.com/avivklas/plexus-search/pkg/searcher"
)

func setupTestServer() (*api.Server, *searcher.Searcher) {
	ds := docstore.New()
	is := indexstore.New()
	s := searcher.New(ds, is)
	srv, _ := api.NewServer("127.0.0.1:0", s)
	return srv, s
}

func TestAPIDocumentLifecycle(t *testing.T) {
	srv, _ := setupTestServer()
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	client := ts.Client()

	// 1. Index document (raw JSON)
	docBody := []byte(`{"title":"Modern Distributed Systems","author":"Alex Petrov","year":2022}`)
	resp, err := client.Post(ts.URL+"/api/v1/indexes/books/docs?id=book-101", "application/json", bytes.NewReader(docBody))
	if err != nil {
		t.Fatalf("POST doc failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected status 201, got %d", resp.StatusCode)
	}

	var created map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&created); err != nil {
		t.Fatalf("decode created response: %v", err)
	}
	if created["id"] != "book-101" || created["index"] != "books" {
		t.Fatalf("unexpected created response: %+v", created)
	}

	// 2. Fetch document
	getResp, err := client.Get(ts.URL + "/api/v1/indexes/books/docs/book-101")
	if err != nil {
		t.Fatalf("GET doc failed: %v", err)
	}
	defer getResp.Body.Close()

	if getResp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", getResp.StatusCode)
	}

	var fetched map[string]any
	if err := json.NewDecoder(getResp.Body).Decode(&fetched); err != nil {
		t.Fatalf("decode fetched doc: %v", err)
	}
	if fetched["id"] != "book-101" {
		t.Fatalf("unexpected fetched doc: %+v", fetched)
	}

	// 3. Search document
	searchQuery := []byte(`{
		"query": {
			"type": "match",
			"field": "title",
			"value": "distributed"
		},
		"load_docs": true
	}`)
	searchResp, err := client.Post(ts.URL+"/api/v1/indexes/books/search", "application/json", bytes.NewReader(searchQuery))
	if err != nil {
		t.Fatalf("search failed: %v", err)
	}
	defer searchResp.Body.Close()

	if searchResp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200 on search, got %d", searchResp.StatusCode)
	}

	var searchResult indexstore.SearchResult
	if err := json.NewDecoder(searchResp.Body).Decode(&searchResult); err != nil {
		t.Fatalf("decode search result: %v", err)
	}
	if searchResult.Total != 1 || len(searchResult.Hits) != 1 {
		t.Fatalf("expected 1 hit, got total=%d", searchResult.Total)
	}
	if searchResult.Hits[0].ID != "book-101" {
		t.Fatalf("expected hit id book-101, got %s", searchResult.Hits[0].ID)
	}

	// 4. Cluster status
	statusResp, err := client.Get(ts.URL + "/api/v1/cluster/status")
	if err != nil {
		t.Fatalf("GET status failed: %v", err)
	}
	defer statusResp.Body.Close()

	if statusResp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200 on cluster status, got %d", statusResp.StatusCode)
	}

	var status searcher.ClusterStatus
	if err := json.NewDecoder(statusResp.Body).Decode(&status); err != nil {
		t.Fatalf("decode status: %v", err)
	}
	if status.Indexes["books"].DocCount != 1 {
		t.Fatalf("expected books doc count 1 in cluster status, got %d", status.Indexes["books"].DocCount)
	}

	// 5. Delete document
	req, _ := http.NewRequest(http.MethodDelete, ts.URL+"/api/v1/indexes/books/docs/book-101", nil)
	delResp, err := client.Do(req)
	if err != nil {
		t.Fatalf("DELETE doc failed: %v", err)
	}
	defer delResp.Body.Close()

	if delResp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200 on delete, got %d", delResp.StatusCode)
	}

	// 6. Verify 404
	get404Resp, err := client.Get(ts.URL + "/api/v1/indexes/books/docs/book-101")
	if err != nil {
		t.Fatalf("GET doc failed: %v", err)
	}
	defer get404Resp.Body.Close()

	if get404Resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", get404Resp.StatusCode)
	}
}

func TestAPIBatchAndEnvelopeIndexing(t *testing.T) {
	srv, _ := setupTestServer()
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	client := ts.Client()

	// Envelope format
	envDoc := []byte(`{
		"id": "env-1",
		"data": {"title": "Envelope Title", "score": 9.9},
		"metadata": {"source": "crawler"}
	}`)

	resp, err := client.Post(ts.URL+"/api/v1/indexes/news/docs", "application/json", bytes.NewReader(envDoc))
	if err != nil {
		t.Fatalf("POST envelope doc failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected status 201, got %d", resp.StatusCode)
	}

	// Batch indexing
	batchBody := []byte(`{
		"indexes": [
			{"id": "env-2", "data": {"title": "Batch News 2", "score": 8.5}},
			{"id": "env-3", "data": {"title": "Batch News 3", "score": 7.0}}
		]
	}`)
	batchResp, err := client.Post(ts.URL+"/api/v1/indexes/news/batch", "application/json", bytes.NewReader(batchBody))
	if err != nil {
		t.Fatalf("POST batch failed: %v", err)
	}
	defer batchResp.Body.Close()

	if batchResp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", batchResp.StatusCode)
	}

	// List indexes
	listResp, err := client.Get(ts.URL + "/api/v1/indexes")
	if err != nil {
		t.Fatalf("GET indexes failed: %v", err)
	}
	defer listResp.Body.Close()

	var indexesRes map[string]any
	if err := json.NewDecoder(listResp.Body).Decode(&indexesRes); err != nil {
		t.Fatalf("decode list indexes: %v", err)
	}
	idxMap, ok := indexesRes["indexes"].(map[string]any)
	if !ok || idxMap["news"] == nil {
		t.Fatalf("expected news index in list: %+v", indexesRes)
	}
}
