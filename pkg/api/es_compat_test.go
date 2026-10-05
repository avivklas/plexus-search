package api_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestESCompatAPI(t *testing.T) {
	srv, _ := setupTestServer()
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	client := ts.Client()

	// 1. Root info (GET /)
	resp, err := client.Get(ts.URL + "/")
	if err != nil {
		t.Fatalf("GET / failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	var rootInfo map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&rootInfo); err != nil {
		t.Fatalf("decode root info: %v", err)
	}
	if rootInfo["tagline"] != "You Know, for Search" {
		t.Fatalf("unexpected tagline: %v", rootInfo["tagline"])
	}

	// 2. Cluster health (GET /_cluster/health)
	healthResp, err := client.Get(ts.URL + "/_cluster/health")
	if err != nil {
		t.Fatalf("GET /_cluster/health failed: %v", err)
	}
	defer healthResp.Body.Close()
	if healthResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", healthResp.StatusCode)
	}
	var health map[string]any
	if err := json.NewDecoder(healthResp.Body).Decode(&health); err != nil {
		t.Fatalf("decode health: %v", err)
	}
	if health["status"] != "green" {
		t.Fatalf("expected status green, got %v", health["status"])
	}

	// 3. Create index (PUT /articles)
	req, _ := http.NewRequest(http.MethodPut, ts.URL+"/articles", nil)
	putResp, err := client.Do(req)
	if err != nil {
		t.Fatalf("PUT /articles failed: %v", err)
	}
	defer putResp.Body.Close()
	if putResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", putResp.StatusCode)
	}

	// 4. Bulk indexing (POST /articles/_bulk)
	bulkData := `{"index": {"_index": "articles", "_id": "art-1"}}
{"title": "Distributed Consensus in Plexus", "category": "engineering", "views": 1500}
{"index": {"_index": "articles", "_id": "art-2"}}
{"title": "Search Engines Internals", "category": "engineering", "views": 2500}
{"index": {"_index": "articles", "_id": "art-3"}}
{"title": "Cooking Recipes", "category": "cooking", "views": 300}
`
	bulkResp, err := client.Post(ts.URL+"/articles/_bulk", "application/x-ndjson", bytes.NewBufferString(bulkData))
	if err != nil {
		t.Fatalf("POST /articles/_bulk failed: %v", err)
	}
	defer bulkResp.Body.Close()
	if bulkResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", bulkResp.StatusCode)
	}
	var bulkResult map[string]any
	if err := json.NewDecoder(bulkResp.Body).Decode(&bulkResult); err != nil {
		t.Fatalf("decode bulk result: %v", err)
	}
	if bulkResult["errors"] != false {
		t.Fatalf("expected errors false, got %v", bulkResult["errors"])
	}
	items, ok := bulkResult["items"].([]any)
	if !ok || len(items) != 3 {
		t.Fatalf("expected 3 items, got %d", len(items))
	}

	// 5. Refresh (POST /articles/_refresh)
	refResp, err := client.Post(ts.URL+"/articles/_refresh", "application/json", nil)
	if err != nil {
		t.Fatalf("POST /articles/_refresh failed: %v", err)
	}
	defer refResp.Body.Close()
	if refResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", refResp.StatusCode)
	}

	// 6. Check index exists (HEAD /articles)
	headReq, _ := http.NewRequest(http.MethodHead, ts.URL+"/articles", nil)
	headResp, err := client.Do(headReq)
	if err != nil {
		t.Fatalf("HEAD /articles failed: %v", err)
	}
	defer headResp.Body.Close()
	if headResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", headResp.StatusCode)
	}

	// 7. Get Document (GET /articles/_doc/art-1)
	docResp, err := client.Get(ts.URL + "/articles/_doc/art-1")
	if err != nil {
		t.Fatalf("GET /articles/_doc/art-1 failed: %v", err)
	}
	defer docResp.Body.Close()
	if docResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", docResp.StatusCode)
	}
	var docData map[string]any
	if err := json.NewDecoder(docResp.Body).Decode(&docData); err != nil {
		t.Fatalf("decode doc: %v", err)
	}
	if docData["found"] != true || docData["_id"] != "art-1" {
		t.Fatalf("unexpected doc: %+v", docData)
	}

	// 8. Search query: match (POST /articles/_search)
	searchBody := []byte(`{
		"query": {
			"match": {"title": "Consensus"}
		}
	}`)
	searchResp, err := client.Post(ts.URL+"/articles/_search", "application/json", bytes.NewReader(searchBody))
	if err != nil {
		t.Fatalf("POST /articles/_search failed: %v", err)
	}
	defer searchResp.Body.Close()
	if searchResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", searchResp.StatusCode)
	}
	var searchResult struct {
		Hits struct {
			Total struct {
				Value int `json:"value"`
			} `json:"total"`
			Hits []struct {
				ID     string         `json:"_id"`
				Source map[string]any `json:"_source"`
			} `json:"hits"`
		} `json:"hits"`
	}
	if err := json.NewDecoder(searchResp.Body).Decode(&searchResult); err != nil {
		t.Fatalf("decode search result: %v", err)
	}
	if searchResult.Hits.Total.Value != 1 {
		t.Fatalf("expected 1 hit, got %d", searchResult.Hits.Total.Value)
	}
	if searchResult.Hits.Hits[0].ID != "art-1" {
		t.Fatalf("expected art-1, got %s", searchResult.Hits.Hits[0].ID)
	}

	// 9. Search query: bool with must and filter
	boolBody := []byte(`{
		"query": {
			"bool": {
				"must": [{"match": {"title": "Search"}}],
				"filter": [{"term": {"category": "engineering"}}]
			}
		}
	}`)
	boolResp, err := client.Post(ts.URL+"/articles/_search", "application/json", bytes.NewReader(boolBody))
	if err != nil {
		t.Fatalf("POST /articles/_search bool query failed: %v", err)
	}
	defer boolResp.Body.Close()
	if boolResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", boolResp.StatusCode)
	}
	var boolResult struct {
		Hits struct {
			Total struct {
				Value int `json:"value"`
			} `json:"total"`
			Hits []struct {
				ID string `json:"_id"`
			} `json:"hits"`
		} `json:"hits"`
	}
	if err := json.NewDecoder(boolResp.Body).Decode(&boolResult); err != nil {
		t.Fatalf("decode bool result: %v", err)
	}
	if boolResult.Hits.Total.Value != 1 || boolResult.Hits.Hits[0].ID != "art-2" {
		t.Fatalf("unexpected bool result: %+v", boolResult)
	}

	// 10. Node stats (GET /_nodes/stats)
	statsResp, err := client.Get(ts.URL + "/_nodes/stats")
	if err != nil {
		t.Fatalf("GET /_nodes/stats failed: %v", err)
	}
	defer statsResp.Body.Close()
	if statsResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", statsResp.StatusCode)
	}
}
