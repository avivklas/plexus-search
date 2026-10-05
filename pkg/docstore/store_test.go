package docstore_test

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/avivklas/plexus-search/pkg/docstore"
)

func TestDocStoreCRUD(t *testing.T) {
	ctx := context.Background()
	store := docstore.New()

	if store.ID() != "docstore" {
		t.Fatalf("expected store ID 'docstore', got %s", store.ID())
	}

	// 1. Put document
	rawDoc := json.RawMessage(`{"title":"Designing Data-Intensive Applications","author":"Martin Kleppmann","year":2017}`)
	meta := map[string]string{"category": "tech"}
	doc, err := store.Put(ctx, docstore.PutDocRequest{
		Index:    "books",
		ID:       "book-1",
		Data:     rawDoc,
		Metadata: meta,
	})
	if err != nil {
		t.Fatalf("put failed: %v", err)
	}

	if doc.ID != "book-1" || doc.Index != "books" || doc.Revision != 1 {
		t.Fatalf("unexpected doc fields: %+v", doc)
	}
	if doc.Metadata["category"] != "tech" {
		t.Fatalf("expected metadata category tech, got %v", doc.Metadata)
	}

	// 2. Get document (local read)
	fetched, found := store.Get("books", "book-1")
	if !found || fetched == nil {
		t.Fatalf("document not found on local get")
	}
	if fetched.Revision != 1 || string(fetched.Data) != string(rawDoc) {
		t.Fatalf("fetched doc mismatch: %+v", fetched)
	}

	// 3. Update document (revision increments)
	updatedRaw := json.RawMessage(`{"title":"Designing Data-Intensive Applications 2nd Ed"}`)
	doc2, err := store.Put(ctx, docstore.PutDocRequest{
		Index: "books",
		ID:    "book-1",
		Data:  updatedRaw,
	})
	if err != nil {
		t.Fatalf("update failed: %v", err)
	}
	if doc2.Revision != 2 {
		t.Fatalf("expected revision 2 after update, got %d", doc2.Revision)
	}

	// 4. Delete document
	deleted, err := store.Delete(ctx, docstore.DeleteDocRequest{
		Index: "books",
		ID:    "book-1",
	})
	if err != nil {
		t.Fatalf("delete failed: %v", err)
	}
	if !deleted {
		t.Fatalf("expected deleted to be true")
	}

	// 5. Verify gone
	_, foundAfter := store.Get("books", "book-1")
	if foundAfter {
		t.Fatalf("document still found after deletion")
	}
}

func TestDocStoreBatch(t *testing.T) {
	ctx := context.Background()
	store := docstore.New()

	req := docstore.BatchDocsRequest{
		Index: "articles",
		Puts: []docstore.PutDocRequest{
			{ID: "art-1", Data: json.RawMessage(`{"title":"Article 1"}`)},
			{ID: "art-2", Data: json.RawMessage(`{"title":"Article 2"}`)},
			{ID: "art-3", Data: json.RawMessage(`{"title":"Article 3"}`)},
		},
	}

	resp, err := store.Batch(ctx, req)
	if err != nil {
		t.Fatalf("batch failed: %v", err)
	}
	if resp.PutCount != 3 {
		t.Fatalf("expected 3 puts, got %d", resp.PutCount)
	}

	if store.Count("articles") != 3 {
		t.Fatalf("expected 3 articles, got %d", store.Count("articles"))
	}

	// Batch with deletes and puts
	batch2 := docstore.BatchDocsRequest{
		Index: "articles",
		Puts: []docstore.PutDocRequest{
			{ID: "art-4", Data: json.RawMessage(`{"title":"Article 4"}`)},
		},
		Deletes: []docstore.DeleteDocRequest{
			{Index: "articles", ID: "art-1"},
			{Index: "articles", ID: "art-2"},
		},
	}

	resp2, err := store.Batch(ctx, batch2)
	if err != nil {
		t.Fatalf("batch2 failed: %v", err)
	}
	if resp2.PutCount != 1 || resp2.DeleteCount != 2 {
		t.Fatalf("expected 1 put and 2 deletes, got %+v", resp2)
	}

	if store.Count("articles") != 2 {
		t.Fatalf("expected 2 remaining articles, got %d", store.Count("articles"))
	}
}

func TestDocStoreSnapshotRestore(t *testing.T) {
	ctx := context.Background()
	s1 := docstore.New()

	for i := 1; i <= 5; i++ {
		_, err := s1.Put(ctx, docstore.PutDocRequest{
			Index: "items",
			ID:    fmt.Sprintf("%d", i),
			Data:  json.RawMessage(fmt.Sprintf(`{"val": %d}`, i)),
		})
		if err != nil {
			t.Fatalf("put failed: %v", err)
		}
	}

	snap, err := s1.Snapshot()
	if err != nil {
		t.Fatalf("snapshot failed: %v", err)
	}

	s2 := docstore.New()
	if err := s2.Restore(snap); err != nil {
		t.Fatalf("restore failed: %v", err)
	}

	if s2.Count("items") != 5 {
		t.Fatalf("expected 5 items in restored store, got %d", s2.Count("items"))
	}

	doc, found := s2.Get("items", "3")
	if !found || doc == nil {
		t.Fatalf("expected item 3 to be present in restored store")
	}
}
