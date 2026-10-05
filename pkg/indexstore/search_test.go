package indexstore_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/avivklas/plexus-search/pkg/indexstore"
)

func floatPtr(f float64) *float64 { return &f }
func boolPtr(b bool) *bool       { return &b }

func TestIndexStoreQueries(t *testing.T) {
	ctx := context.Background()
	store := indexstore.New()
	defer store.Close()

	if store.ID() != "indexstore" {
		t.Fatalf("expected store ID 'indexstore', got %s", store.ID())
	}

	// 1. Index sample documents
	docs := []struct {
		id   string
		data map[string]any
	}{
		{
			id: "doc-1",
			data: map[string]any{
				"title":       "Distributed Systems with Raft Consensus",
				"description": "Explains how consensus algorithms like Raft work in distributed state machines.",
				"category":    "databases",
				"tags":        []string{"distributed", "raft", "consensus"},
				"stars":       150.0,
				"active":      true,
			},
		},
		{
			id: "doc-2",
			data: map[string]any{
				"title":       "Full-Text Search Engine Architecture",
				"description": "Building inverted indexes and tokenizers for fast microsecond queries.",
				"category":    "search",
				"tags":        []string{"search", "bleve", "indexing"},
				"stars":       95.0,
				"active":      true,
			},
		},
		{
			id: "doc-3",
			data: map[string]any{
				"title":       "Eliminating CDC Lag with Plexus",
				"description": "Atomic multi-store state machines solve dual-write and eventual consistency lag.",
				"category":    "databases",
				"tags":        []string{"plexus", "cdc", "raft"},
				"stars":       320.0,
				"active":      false,
			},
		},
	}

	for _, d := range docs {
		raw, _ := json.Marshal(d.data)
		res, err := store.Index(ctx, indexstore.IndexDocRequest{
			Index: "articles",
			ID:    d.id,
			Data:  raw,
		})
		if err != nil {
			t.Fatalf("index doc %s failed: %v", d.id, err)
		}
		if !res.Success {
			t.Fatalf("expected success for doc %s", d.id)
		}
	}

	count, err := store.DocCount("articles")
	if err != nil || count != 3 {
		t.Fatalf("expected doc count 3, got count=%d, err=%v", count, err)
	}

	// 2. Test Match Query
	t.Run("MatchQuery", func(t *testing.T) {
		res, err := store.SearchMatch("articles", "title", "systems", 0, 10)
		if err != nil {
			t.Fatalf("match search failed: %v", err)
		}
		if res.Total != 1 || len(res.Hits) != 1 || res.Hits[0].ID != "doc-1" {
			t.Fatalf("expected doc-1, got total=%d hits=%+v", res.Total, res.Hits)
		}
	})

	// 3. Test MatchPhrase Query
	t.Run("MatchPhraseQuery", func(t *testing.T) {
		res, err := store.SearchMatchPhrase("articles", "description", "inverted indexes", 0, 10)
		if err != nil {
			t.Fatalf("phrase search failed: %v", err)
		}
		if res.Total != 1 || len(res.Hits) != 1 || res.Hits[0].ID != "doc-2" {
			t.Fatalf("expected doc-2, got total=%d hits=%+v", res.Total, res.Hits)
		}
	})

	// 4. Test Term Query
	t.Run("TermQuery", func(t *testing.T) {
		res, err := store.SearchTerm("articles", "category", "databases", 0, 10)
		if err != nil {
			t.Fatalf("term search failed: %v", err)
		}
		if res.Total != 2 {
			t.Fatalf("expected 2 hits for category:databases, got %d", res.Total)
		}
	})

	// 5. Test Prefix Query
	t.Run("PrefixQuery", func(t *testing.T) {
		res, err := store.SearchPrefix("articles", "category", "data", 0, 10)
		if err != nil {
			t.Fatalf("prefix search failed: %v", err)
		}
		if res.Total != 2 {
			t.Fatalf("expected 2 hits for prefix 'data', got %d", res.Total)
		}
	})

	// 6. Test Fuzzy Query
	t.Run("FuzzyQuery", func(t *testing.T) {
		// "architcture" misspelling of "architecture"
		res, err := store.SearchFuzzy("articles", "title", "architcture", 2, 0, 10)
		if err != nil {
			t.Fatalf("fuzzy search failed: %v", err)
		}
		if res.Total != 1 || res.Hits[0].ID != "doc-2" {
			t.Fatalf("expected doc-2 for fuzzy match, got total=%d hits=%+v", res.Total, res.Hits)
		}
	})

	// 7. Test Numeric Range Query
	t.Run("NumericRangeQuery", func(t *testing.T) {
		// stars between 100 and 400 (should match doc-1: 150, doc-3: 320)
		res, err := store.SearchNumericRange("articles", "stars", floatPtr(100.0), floatPtr(400.0), boolPtr(true), boolPtr(true), 0, 10)
		if err != nil {
			t.Fatalf("numeric range failed: %v", err)
		}
		if res.Total != 2 {
			t.Fatalf("expected 2 hits in range [100, 400], got %d", res.Total)
		}
	})

	// 8. Test Boolean Query (Must + MustNot)
	t.Run("BooleanQuery", func(t *testing.T) {
		must := []indexstore.QueryDefinition{
			{Type: indexstore.QueryTypeTerm, Field: "category", Value: "databases"},
		}
		mustNot := []indexstore.QueryDefinition{
			{Type: indexstore.QueryTypeMatch, Field: "title", Value: "CDC"},
		}
		res, err := store.SearchBoolean("articles", must, nil, mustNot, 0, 10)
		if err != nil {
			t.Fatalf("boolean search failed: %v", err)
		}
		if res.Total != 1 || res.Hits[0].ID != "doc-1" {
			t.Fatalf("expected doc-1 from boolean query, got total=%d hits=%+v", res.Total, res.Hits)
		}
	})

	// 9. Test Faceting and Pagination
	t.Run("FacetingAndPagination", func(t *testing.T) {
		req := indexstore.SearchRequest{
			Query: indexstore.QueryDefinition{Type: indexstore.QueryTypeMatchAll},
			Size:  2,
			From:  0,
			Facets: map[string]indexstore.FacetRequestDef{
				"category_facet": {
					Field: "category",
					Size:  5,
				},
				"stars_range": {
					Field: "stars",
					NumericRanges: []indexstore.NumericRangeDef{
						{Name: "low", Max: floatPtr(100.0)},
						{Name: "high", Min: floatPtr(100.0)},
					},
				},
			},
		}

		res, err := store.Search("articles", req)
		if err != nil {
			t.Fatalf("faceted search failed: %v", err)
		}

		if res.Total != 3 {
			t.Fatalf("expected total 3, got %d", res.Total)
		}
		if len(res.Hits) != 2 {
			t.Fatalf("expected paginated page size 2, got %d", len(res.Hits))
		}

		catFacet, ok := res.Facets["category_facet"]
		if !ok {
			t.Fatalf("missing category_facet in response")
		}
		if len(catFacet.Terms) == 0 {
			t.Fatalf("expected term facet results in category_facet")
		}

		starFacet, ok := res.Facets["stars_range"]
		if !ok {
			t.Fatalf("missing stars_range in response")
		}
		if len(starFacet.Ranges) == 0 {
			t.Fatalf("expected range facet results in stars_range")
		}
	})
}

func TestIndexStoreSnapshotRestore(t *testing.T) {
	ctx := context.Background()
	s1 := indexstore.New()
	defer s1.Close()

	for i := 1; i <= 4; i++ {
		data, _ := json.Marshal(map[string]any{
			"name":  "product",
			"index": i,
		})
		_, err := s1.Index(ctx, indexstore.IndexDocRequest{
			Index: "products",
			ID:    string(rune('A' + i)),
			Data:  data,
		})
		if err != nil {
			t.Fatalf("index failed: %v", err)
		}
	}

	snap, err := s1.Snapshot()
	if err != nil {
		t.Fatalf("snapshot failed: %v", err)
	}

	s2 := indexstore.New()
	defer s2.Close()

	if err := s2.Restore(snap); err != nil {
		t.Fatalf("restore failed: %v", err)
	}

	count, err := s2.DocCount("products")
	if err != nil || count != 4 {
		t.Fatalf("expected 4 docs in restored index, got count=%d, err=%v", count, err)
	}

	res, err := s2.SearchMatch("products", "name", "product", 0, 10)
	if err != nil {
		t.Fatalf("search on restored index failed: %v", err)
	}
	if res.Total != 4 {
		t.Fatalf("expected 4 search hits on restored index, got %d", res.Total)
	}
}
