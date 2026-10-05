package searcher

import (
	"encoding/json"
	"time"

	"github.com/avivklas/plexus"
	"github.com/avivklas/plexus-search/pkg/docstore"
)

const (
	CmdAtomicIndex  plexus.CommandType = "searcher.atomic_index"
	CmdAtomicDelete plexus.CommandType = "searcher.atomic_delete"
	CmdAtomicBatch  plexus.CommandType = "searcher.atomic_batch"
)

// AtomicIndexRequest describes a document to be indexed atomically into both DocStore and IndexStore.
type AtomicIndexRequest struct {
	Index    string            `json:"index"`
	ID       string            `json:"id"`
	Data     json.RawMessage   `json:"data"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

// AtomicIndexResponse contains the resulting Document metadata.
type AtomicIndexResponse struct {
	Document *docstore.Document `json:"document"`
}

// AtomicDeleteRequest describes a document to be deleted atomically from both DocStore and IndexStore.
type AtomicDeleteRequest struct {
	Index string `json:"index"`
	ID    string `json:"id"`
}

// AtomicDeleteResponse contains the deletion status.
type AtomicDeleteResponse struct {
	Index   string `json:"index"`
	ID      string `json:"id"`
	Deleted bool   `json:"deleted"`
}

// AtomicBatchRequest describes batch mutations across DocStore and IndexStore.
type AtomicBatchRequest struct {
	Index   string                `json:"index"`
	Indexes []AtomicIndexRequest  `json:"indexes,omitempty"`
	Deletes []AtomicDeleteRequest `json:"deletes,omitempty"`
}

// AtomicBatchResponse contains the counts of batch operations applied.
type AtomicBatchResponse struct {
	Index        string `json:"index"`
	IndexedCount int    `json:"indexed_count"`
	DeletedCount int    `json:"deleted_count"`
}

// ClusterNodeInfo contains information about a single cluster node.
type ClusterNodeInfo struct {
	ID      string `json:"id"`
	Address string `json:"address"`
	Voter   bool   `json:"voter"`
	Leader  bool   `json:"leader"`
}

// ClusterStatus contains health, leadership, consensus, and index statistics.
type ClusterStatus struct {
	NodeID         string                     `json:"node_id"`
	IsLeader       bool                       `json:"is_leader"`
	LeaderID       string                     `json:"leader_id,omitempty"`
	LeaderAddress  string                     `json:"leader_address,omitempty"`
	RaftState      string                     `json:"raft_state"`
	CommittedIndex uint64                     `json:"committed_index"`
	AppliedIndex   uint64                     `json:"applied_index"`
	Indexes        map[string]IndexStatistics `json:"indexes"`
	Nodes          []ClusterNodeInfo          `json:"nodes"`
	Uptime         string                     `json:"uptime"`
}

// IndexStatistics contains document counts and metrics for a named index.
type IndexStatistics struct {
	DocCount   int       `json:"doc_count"`
	LastUpdate time.Time `json:"last_update,omitempty"`
}
