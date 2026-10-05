package searcher_test

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/avivklas/plexus"
	"github.com/avivklas/plexus-search/pkg/docstore"
	"github.com/avivklas/plexus-search/pkg/indexstore"
	"github.com/avivklas/plexus-search/pkg/searcher"
	"github.com/hashicorp/raft"
)

func setupTestCluster(t *testing.T, nodeID string) (*plexus.Cluster, *searcher.Searcher) {
	tmpDir, err := os.MkdirTemp("", "plexus-searcher-test-*")
	if err != nil {
		t.Fatalf("create temp dir: %v", err)
	}
	t.Cleanup(func() { os.RemoveAll(tmpDir) })

	cfg := plexus.DefaultClusterConfig(nodeID, "127.0.0.1:9200", tmpDir)
	cfg.Bootstrap = true
	cfg.ApplyTimeout = 5 * time.Second

	c, err := plexus.NewCluster(cfg)
	if err != nil {
		t.Fatalf("NewCluster failed: %v", err)
	}

	_, trans := raft.NewInmemTransport(raft.ServerAddress("127.0.0.1:9200"))
	c.DefaultRaftMachine().WithCustomStores(
		raft.NewInmemStore(),
		raft.NewInmemStore(),
		raft.NewInmemSnapshotStore(),
		trans,
	)

	ds := docstore.New()
	is := indexstore.New()
	s := searcher.New(ds, is)
	s.Register(c)

	if err := c.Start(context.Background()); err != nil {
		t.Fatalf("Cluster Start failed: %v", err)
	}
	t.Cleanup(func() { _ = c.Stop(); _ = is.Close() })

	// Wait for leadership
	deadline := time.Now().Add(5 * time.Second)
	for !c.DefaultMachine().IsLeader() {
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for leader")
		}
		time.Sleep(20 * time.Millisecond)
	}

	return c, s
}

func TestSearcherAtomicIndexingAndZeroCDCLag(t *testing.T) {
	_, s := setupTestCluster(t, "node-1")
	ctx := context.Background()

	// 1. Atomically index a document through Raft consensus
	docData := json.RawMessage(`{"title":"Distributed Raft Consensus Engine","author":"Leslie Lamport","stars":1000}`)
	meta := map[string]string{"env": "prod"}

	indexedDoc, err := s.IndexDoc(ctx, "papers", "paper-1", docData, meta)
	if err != nil {
		t.Fatalf("IndexDoc failed: %v", err)
	}

	if indexedDoc.ID != "paper-1" || indexedDoc.Revision != 1 {
		t.Fatalf("unexpected indexed doc: %+v", indexedDoc)
	}

	// 2. Broken CAP verification: Instant zero-hop local in-memory read
	fetchedDoc, err := s.GetDoc(ctx, "papers", "paper-1")
	if err != nil {
		t.Fatalf("GetDoc failed: %v", err)
	}
	if fetchedDoc.Revision != 1 || string(fetchedDoc.Data) != string(docData) {
		t.Fatalf("fetched doc mismatch: %+v", fetchedDoc)
	}

	// 3. ZERO CDC LAG verification: Inverted index is immediately searchable
	searchRes, err := s.Search(ctx, "papers", indexstore.SearchRequest{
		Query:    indexstore.QueryDefinition{Type: indexstore.QueryTypeMatch, Field: "title", Value: "consensus"},
		LoadDocs: true,
	})
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	if searchRes.Total != 1 || len(searchRes.Hits) != 1 {
		t.Fatalf("expected 1 hit with zero CDC lag, got total=%d", searchRes.Total)
	}

	hit := searchRes.Hits[0]
	if hit.ID != "paper-1" {
		t.Fatalf("expected hit id paper-1, got %s", hit.ID)
	}

	// Verify LoadDocs hydrated the full document
	if hit.Document == nil {
		t.Fatalf("expected hydrated document in search hit")
	}
	hydratedDoc, ok := hit.Document.(*docstore.Document)
	if !ok || hydratedDoc.ID != "paper-1" {
		t.Fatalf("expected hydrated *docstore.Document, got %T: %+v", hit.Document, hit.Document)
	}

	// 4. Test Atomic Deletion
	if err := s.DeleteDoc(ctx, "papers", "paper-1"); err != nil {
		t.Fatalf("DeleteDoc failed: %v", err)
	}

	// Verify immediately gone from docstore
	_, err = s.GetDoc(ctx, "papers", "paper-1")
	if err != searcher.ErrNotFound {
		t.Fatalf("expected ErrNotFound after deletion, got %v", err)
	}

	// Verify immediately gone from index
	afterDelRes, err := s.Search(ctx, "papers", indexstore.SearchRequest{
		Query: indexstore.QueryDefinition{Type: indexstore.QueryTypeMatchAll},
	})
	if err != nil {
		t.Fatalf("Search after delete failed: %v", err)
	}
	if afterDelRes.Total != 0 {
		t.Fatalf("expected 0 hits after deletion, got %d", afterDelRes.Total)
	}
}

func TestSearcherBatch(t *testing.T) {
	_, s := setupTestCluster(t, "node-batch")
	ctx := context.Background()

	batchReq := searcher.AtomicBatchRequest{
		Index: "catalog",
		Indexes: []searcher.AtomicIndexRequest{
			{ID: "item-1", Data: json.RawMessage(`{"name":"Mechanical Keyboard","price":120.0}`)},
			{ID: "item-2", Data: json.RawMessage(`{"name":"Wireless Mouse","price":60.0}`)},
			{ID: "item-3", Data: json.RawMessage(`{"name":"4K Monitor","price":450.0}`)},
		},
	}

	batchResp, err := s.Batch(ctx, batchReq)
	if err != nil {
		t.Fatalf("Batch failed: %v", err)
	}
	if batchResp.IndexedCount != 3 {
		t.Fatalf("expected 3 items indexed, got %d", batchResp.IndexedCount)
	}

	// Search items
	res, err := s.Search(ctx, "catalog", indexstore.SearchRequest{
		Query: indexstore.QueryDefinition{Type: indexstore.QueryTypeMatchAll},
	})
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
	if res.Total != 3 {
		t.Fatalf("expected 3 items in catalog, got %d", res.Total)
	}

	// Cluster Status check
	status := s.ClusterStatus()
	if !status.IsLeader {
		t.Fatalf("expected IsLeader true in cluster status")
	}
	if status.Indexes["catalog"].DocCount != 3 {
		t.Fatalf("expected catalog doc count 3 in status, got %d", status.Indexes["catalog"].DocCount)
	}
}
