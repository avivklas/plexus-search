package test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/avivklas/plexus"
	"github.com/avivklas/plexus/pkg/machine"
	"github.com/avivklas/plexus-search/pkg/docstore"
	"github.com/avivklas/plexus-search/pkg/indexstore"
	"github.com/avivklas/plexus-search/pkg/searcher"
	"github.com/hashicorp/raft"
)

type testNode struct {
	cluster  *plexus.Cluster
	searcher *searcher.Searcher
	docStore *docstore.Store
	idxStore *indexstore.Store
	trans    *raft.InmemTransport
}

func setupThreeNodeCluster(t *testing.T) ([]*testNode, func()) {
	tmpDir, err := os.MkdirTemp("", "plexus-replication-test-*")
	if err != nil {
		t.Fatalf("create temp dir: %v", err)
	}

	addrs := []string{"127.0.0.1:9401", "127.0.0.1:9402", "127.0.0.1:9403"}
	transports := make([]*raft.InmemTransport, 3)
	for i := 0; i < 3; i++ {
		_, tr := raft.NewInmemTransport(raft.ServerAddress(addrs[i]))
		transports[i] = tr
	}

	// Fully connect all transports
	for i := 0; i < 3; i++ {
		for j := 0; j < 3; j++ {
			if i != j {
				transports[i].Connect(transports[j].LocalAddr(), transports[j])
			}
		}
	}

	nodes := make([]*testNode, 3)

	for i := 0; i < 3; i++ {
		nodeID := string(rune('1' + i))
		cfg := plexus.DefaultClusterConfig("node-"+nodeID, addrs[i], filepath.Join(tmpDir, "node-"+nodeID))
		cfg.Bootstrap = (i == 0) // Only node-1 bootstraps
		cfg.ApplyTimeout = 5 * time.Second

		c, err := plexus.NewCluster(cfg)
		if err != nil {
			t.Fatalf("NewCluster node-%s failed: %v", nodeID, err)
		}

		c.DefaultRaftMachine().WithCustomStores(
			raft.NewInmemStore(),
			raft.NewInmemStore(),
			raft.NewInmemSnapshotStore(),
			transports[i],
		)

		ds := docstore.New()
		is := indexstore.New()
		s := searcher.New(ds, is)
		s.Register(c)

		nodes[i] = &testNode{
			cluster:  c,
			searcher: s,
			docStore: ds,
			idxStore: is,
			trans:    transports[i],
		}
	}

	// 1. Start leader node-1
	ctx := context.Background()
	if err := nodes[0].cluster.Start(ctx); err != nil {
		t.Fatalf("Start node 0 failed: %v", err)
	}

	// Wait for node-1 to be elected leader
	leaderDeadline := time.Now().Add(5 * time.Second)
	for !nodes[0].cluster.DefaultMachine().IsLeader() {
		if time.Now().After(leaderDeadline) {
			t.Fatalf("timed out waiting for node-1 to become leader")
		}
		time.Sleep(20 * time.Millisecond)
	}

	// 2. Start follower nodes and join them
	for i := 1; i < 3; i++ {
		if err := nodes[i].cluster.Start(ctx); err != nil {
			t.Fatalf("Start node %d failed: %v", i, err)
		}

		nodeID := string(rune('1' + i))
		_, err := nodes[0].cluster.DefaultRaftMachine().HandleJoin(&machine.JoinRequest{
			Node: &machine.Node{
				ID:      "node-" + nodeID,
				Address: string(nodes[i].trans.LocalAddr()),
				Voter:   true,
			},
		})
		if err != nil {
			t.Fatalf("Join node %d failed: %v", i, err)
		}
	}

	// Wait for configuration to stabilize (all 3 members active)
	time.Sleep(200 * time.Millisecond)

	teardown := func() {
		for _, n := range nodes {
			_ = n.cluster.Stop()
			_ = n.idxStore.Close()
		}
		_ = os.RemoveAll(tmpDir)
	}

	return nodes, teardown
}

func TestMultiNodeReplicationAndLocalSearch(t *testing.T) {
	nodes, teardown := setupThreeNodeCluster(t)
	defer teardown()

	ctx := context.Background()
	leaderNode := nodes[0]
	followerNode1 := nodes[1]
	followerNode2 := nodes[2]

	// 1. Write documents to Leader (node-1)
	docs := []struct {
		id   string
		data map[string]any
	}{
		{
			id: "doc-repl-1",
			data: map[string]any{
				"title":    "Consensus in Distributed Databases",
				"category": "distributed",
				"priority": 1,
			},
		},
		{
			id: "doc-repl-2",
			data: map[string]any{
				"title":    "Bleve Inverted Indexing Secrets",
				"category": "search",
				"priority": 2,
			},
		},
	}

	for _, d := range docs {
		raw, _ := json.Marshal(d.data)
		doc, err := leaderNode.searcher.IndexDoc(ctx, "cluster-docs", d.id, raw, nil)
		if err != nil {
			t.Fatalf("IndexDoc on leader failed: %v", err)
		}
		if doc.ID != d.id {
			t.Fatalf("unexpected doc ID: %s", doc.ID)
		}
	}

	// Allow a moment for Raft replication to commit to followers
	time.Sleep(300 * time.Millisecond)

	// 2. Verify Follower 1 (node-2) has the document in its local DocStore
	docF1, err := followerNode1.searcher.GetDoc(ctx, "cluster-docs", "doc-repl-1")
	if err != nil {
		t.Fatalf("follower 1 GetDoc failed: %v", err)
	}
	if docF1.ID != "doc-repl-1" || docF1.Revision != 1 {
		t.Fatalf("follower 1 unexpected doc: %+v", docF1)
	}

	// 3. Verify Follower 2 (node-3) has indexed the document and can search locally with zero hops
	resF2, err := followerNode2.searcher.Search(ctx, "cluster-docs", indexstore.SearchRequest{
		Query:    indexstore.QueryDefinition{Type: indexstore.QueryTypeMatch, Field: "title", Value: "Bleve"},
		LoadDocs: true,
	})
	if err != nil {
		t.Fatalf("follower 2 search failed: %v", err)
	}

	if resF2.Total != 1 || len(resF2.Hits) != 1 {
		t.Fatalf("expected follower 2 to find 1 hit, got total=%d", resF2.Total)
	}
	if resF2.Hits[0].ID != "doc-repl-2" {
		t.Fatalf("expected hit doc-repl-2, got %s", resF2.Hits[0].ID)
	}

	// 4. Verify atomic deletion replicates across the cluster
	if err := leaderNode.searcher.DeleteDoc(ctx, "cluster-docs", "doc-repl-1"); err != nil {
		t.Fatalf("DeleteDoc on leader failed: %v", err)
	}

	time.Sleep(300 * time.Millisecond)

	// Check on follower 1 that it is deleted
	_, err = followerNode1.searcher.GetDoc(ctx, "cluster-docs", "doc-repl-1")
	if err != searcher.ErrNotFound {
		t.Fatalf("expected follower 1 to report not found after delete, got %v", err)
	}

	// Check on follower 2 search that it is gone
	delSearchRes, err := followerNode2.searcher.Search(ctx, "cluster-docs", indexstore.SearchRequest{
		Query: indexstore.QueryDefinition{Type: indexstore.QueryTypeMatch, Field: "title", Value: "Distributed"},
	})
	if err != nil {
		t.Fatalf("follower 2 search after delete failed: %v", err)
	}
	if delSearchRes.Total != 0 {
		t.Fatalf("expected 0 hits on follower 2 after deletion, got %d", delSearchRes.Total)
	}
}
