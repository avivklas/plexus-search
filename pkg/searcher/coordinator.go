package searcher

import (
	"context"
	"fmt"

	"github.com/avivklas/plexus"
	"github.com/avivklas/plexus-search/pkg/docstore"
	"github.com/avivklas/plexus-search/pkg/indexstore"
)

// CoordinatorStore implements plexus.Store to coordinate atomic updates between DocStore and IndexStore.
type CoordinatorStore struct {
	plexus.BaseStore
	docStore   *docstore.Store
	indexStore *indexstore.Store
}

// NewCoordinatorStore creates an initialized CoordinatorStore.
func NewCoordinatorStore(ds *docstore.Store, is *indexstore.Store) *CoordinatorStore {
	cs := &CoordinatorStore{
		BaseStore:  plexus.NewBaseStore(),
		docStore:   ds,
		indexStore: is,
	}

	cs.Handle(CmdAtomicIndex, cs.handleAtomicIndex)
	cs.Handle(CmdAtomicDelete, cs.handleAtomicDelete)
	cs.Handle(CmdAtomicBatch, cs.handleAtomicBatch)

	return cs
}

// ID implements plexus.Store.
func (cs *CoordinatorStore) ID() plexus.StoreID {
	return "searcher"
}

func (cs *CoordinatorStore) handleAtomicIndex(ctx context.Context, req AtomicIndexRequest) (*AtomicIndexResponse, error) {
	// 1. Apply to Document Store
	doc, err := cs.docStore.ApplyPut(ctx, docstore.PutDocRequest{
		Index:    req.Index,
		ID:       req.ID,
		Data:     req.Data,
		Metadata: req.Metadata,
	})
	if err != nil {
		return nil, fmt.Errorf("docstore apply put: %w", err)
	}

	// 2. Apply to Index Store
	if err := cs.indexStore.ApplyIndex(ctx, indexstore.IndexDocRequest{
		Index: req.Index,
		ID:    req.ID,
		Data:  req.Data,
	}); err != nil {
		return nil, fmt.Errorf("indexstore apply index: %w", err)
	}

	return &AtomicIndexResponse{Document: doc}, nil
}

func (cs *CoordinatorStore) handleAtomicDelete(ctx context.Context, req AtomicDeleteRequest) (*AtomicDeleteResponse, error) {
	deleted, err := cs.docStore.ApplyDelete(ctx, docstore.DeleteDocRequest{
		Index: req.Index,
		ID:    req.ID,
	})
	if err != nil {
		return nil, fmt.Errorf("docstore apply delete: %w", err)
	}

	_ = cs.indexStore.ApplyDelete(ctx, indexstore.DeleteDocIndexRequest{
		Index: req.Index,
		ID:    req.ID,
	})

	return &AtomicDeleteResponse{
		Index:   req.Index,
		ID:      req.ID,
		Deleted: deleted,
	}, nil
}

func (cs *CoordinatorStore) handleAtomicBatch(ctx context.Context, req AtomicBatchRequest) (*AtomicBatchResponse, error) {
	var docPuts []docstore.PutDocRequest
	var idxPuts []indexstore.IndexDocRequest

	for _, p := range req.Indexes {
		docPuts = append(docPuts, docstore.PutDocRequest{
			Index:    req.Index,
			ID:       p.ID,
			Data:     p.Data,
			Metadata: p.Metadata,
		})
		idxPuts = append(idxPuts, indexstore.IndexDocRequest{
			Index: req.Index,
			ID:    p.ID,
			Data:  p.Data,
		})
	}

	var docDels []docstore.DeleteDocRequest
	var idxDels []indexstore.DeleteDocIndexRequest

	for _, d := range req.Deletes {
		docDels = append(docDels, docstore.DeleteDocRequest{
			Index: req.Index,
			ID:    d.ID,
		})
		idxDels = append(idxDels, indexstore.DeleteDocIndexRequest{
			Index: req.Index,
			ID:    d.ID,
		})
	}

	docBatchResp, err := cs.docStore.ApplyBatch(ctx, docstore.BatchDocsRequest{
		Index:   req.Index,
		Puts:    docPuts,
		Deletes: docDels,
	})
	if err != nil {
		return nil, fmt.Errorf("docstore apply batch: %w", err)
	}

	_, err = cs.indexStore.ApplyBatch(ctx, indexstore.BatchIndexRequest{
		Index:   req.Index,
		Indexes: idxPuts,
		Deletes: idxDels,
	})
	if err != nil {
		return nil, fmt.Errorf("indexstore apply batch: %w", err)
	}

	return &AtomicBatchResponse{
		Index:        req.Index,
		IndexedCount: docBatchResp.PutCount,
		DeletedCount: docBatchResp.DeleteCount,
	}, nil
}

// Snapshot implements plexus.Store. State is held in DocStore and IndexStore independently.
func (cs *CoordinatorStore) Snapshot() ([]byte, error) {
	return []byte("{}"), nil
}

// Restore implements plexus.Store. State is restored in DocStore and IndexStore independently.
func (cs *CoordinatorStore) Restore(data []byte) error {
	return nil
}
