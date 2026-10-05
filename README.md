# Plexus-Search ⚡🔍

> **Distributed Zero-CDC Search Engine & Document Store powered by Plexus Raft Multi-Store Consensus.**

`plexus-search` embeds a primary Document Store and an in-memory full-text search index ([Bleve](https://github.com/blevesearch/bleve)) inside a single **Multi-Store Raft State Machine**. Every document mutation is replicated across the cluster and applied atomically to both the raw document repository and the Bleve inverted index, completely eliminating the notorious **Dual-Write / CDC Lag Problem**.

---

## The Dual-Write & CDC Lag Problem

In modern data architectures, searching documents typically requires two separate systems: a primary datastore (PostgreSQL, MongoDB, DynamoDB) and an external search cluster (Elasticsearch, OpenSearch).

### The Traditional Approach: Fragile CDC Pipelines

```
[ Application ]
       │  (Write Primary DB)
       ▼
 ┌───────────┐        ┌───────────┐        ┌───────────┐        ┌───────────────┐
 │ PostgreSQL│ ──CDC─►│ Debezium  │ ──────►│   Kafka   │ ──────►│ Elasticsearch │
 └───────────┘ (WAL)  └───────────┘        └───────────┘        └───────────────┘
                                                                        │
                                             Lag: 100ms - 5s ───────────┘
                                             Search misses recent writes!
```

1. **Dual-Write Hazard**: If the app writes to both DB and Elasticsearch directly, network or process crashes cause permanent state divergence.
2. **CDC Ingestion Delay**: Debezium + Kafka + Elasticsearch ingest pipelines introduce 100ms to several seconds of replication lag.
3. **Read-Your-Own-Writes Violation**: A user saves a document and navigates to the search page, but their query returns nothing because the index hasn't caught up.
4. **Operational Nightmare**: Managing Kafka brokers, schema registries, Debezium connectors, and Elasticsearch clusters introduces immense maintenance overhead.

---

## The Plexus-Search Solution

`plexus-search` solves this by placing both the **Document Store** and the **Bleve Inverted Index** into a single Raft-replicated state machine using the [Plexus](https://github.com/avivklas/plexus) framework.

```
                     ┌────────────────────────────────────────────────────────┐
                     │              Plexus-Search Cluster Node                │
                     │                                                        │
[ Write Request ] ──►│   Raft Consensus Engine (Quorum Commit)                │
                     │              │                                         │
                     │              ▼ Single Atomic Apply Step                │
                     │   ┌──────────────────────┬─────────────────────────┐   │
                     │   │   Document Store     │    Bleve Index Store    │   │
                     │   │  (Raw JSON + Revs)   │ (In-Memory Inverted Idx)│   │
                     │   └──────────────────────┴─────────────────────────┘   │
                     │              ▲                        ▲                │
                     └──────────────┼────────────────────────┼────────────────┘
                                    │                        │
[ Zero-Hop Local Read ] ────────────┘                        │
[ Microsecond Search  ] ─────────────────────────────────────┘
```

### Key Guarantees:
- **Zero CDC Lag**: Raft log entries mutate both the document store and the Bleve index in the exact same state machine transition on every replica.
- **Broken CAP Sequential Consistency**: Writes return `201 Created` only after Quorum consensus **and** local state machine application. Consequently, reads and searches against the local node are guaranteed to see the write immediately.
- **Zero Network Hops for Queries**: Searches execute against the in-memory inverted index on the local node with microsecond latency without touching the network.
- **Single Cohesive Binary**: No Kafka, no Debezium, no Elasticsearch cluster.

---

## Architecture Components

```
plexus-search/
├── cmd/plexus-search/      # CLI executable entrypoint
├── pkg/
│   ├── docstore/          # Raw JSON document store implementing plexus.Store
│   ├── indexstore/        # In-memory Bleve inverted index implementing plexus.Store
│   ├── searcher/          # High-level atomic coordinator & local query engine
│   └── api/               # High-performance REST HTTP API
└── test/                  # Multi-node Raft replication integration tests
```

### 1. `pkg/docstore`
- Implements `plexus.Store` with ID `"docstore"`.
- Stores raw JSON documents with monotonic revision numbers, created/updated timestamps, and user metadata.
- Replicated handlers: `CmdPutDoc`, `CmdGetDoc`, `CmdDeleteDoc`, `CmdBatchDocs`.
- Full snapshot and restore support.

### 2. `pkg/indexstore`
- Implements `plexus.Store` with ID `"indexstore"`.
- Manages in-memory Bleve index instances per index namespace.
- Supports rich querying:
  - **Match** (analyzed full-text)
  - **MatchPhrase** (exact sequence phrase match)
  - **Term** (exact unanalyzed term match)
  - **Prefix** (term prefix match)
  - **Fuzzy** (Levenshtein distance matching)
  - **NumericRange** (inclusive/exclusive boundary ranges)
  - **Boolean** (`must`, `should`, `must_not` clauses)
  - **Faceting** (term counts, numeric range facets, date range facets)
  - **Pagination** (`from`, `size`) & Highlighting
- Replicated handlers: `CmdIndexDoc`, `CmdDeleteDocIndex`, `CmdBatchIndex`.

### 3. `pkg/searcher`
- High-level coordinator managing `DocStore` and `IndexStore` on the same Plexus node.
- Exposes `CmdAtomicIndex`, `CmdAtomicDelete`, and `CmdAtomicBatch` to apply mutations atomically to both stores in a single Raft commit.
- Enables enriched search results (`load_docs: true`), hydrating matching Bleve hits with full documents from local memory with zero network hops.

### 4. `pkg/api`
- Clean, idiomatic REST HTTP API with CORS, panic recovery, and JSON marshaling.
- Provides endpoints for document ingestion, fetching, deletion, search, batching, and cluster health.

---

## Building & Running

### Prerequisites
- Go 1.22+ (tested with Go 1.27)

### Build
```bash
go build -o bin/plexus-search ./cmd/plexus-search
```

### Run a Single Standalone Node (Leader)
```bash
./bin/plexus-search \
  --id node-1 \
  --http-addr 127.0.0.1:8080 \
  --raft-addr 127.0.0.1:9000 \
  --data-dir ./data/node-1 \
  --bootstrap
```

### Run a 3-Node Raft Cluster

**Node 1 (Bootstrap Leader):**
```bash
./bin/plexus-search \
  --id node-1 \
  --http-addr 127.0.0.1:8081 \
  --raft-addr 127.0.0.1:9001 \
  --data-dir ./data/node-1 \
  --bootstrap
```

**Node 2 (Follower):**
```bash
./bin/plexus-search \
  --id node-2 \
  --http-addr 127.0.0.1:8082 \
  --raft-addr 127.0.0.1:9002 \
  --data-dir ./data/node-2 \
  --join 127.0.0.1:9001
```

**Node 3 (Follower):**
```bash
./bin/plexus-search \
  --id node-3 \
  --http-addr 127.0.0.1:8083 \
  --raft-addr 127.0.0.1:9003 \
  --data-dir ./data/node-3 \
  --join 127.0.0.1:9001
```

---

## REST API Reference & Examples

### 1. Index a Document
`POST /api/v1/indexes/{index}/docs`

Supports direct JSON payloads or envelope format with optional `?id=` query parameter.

```bash
curl -X POST "http://127.0.0.1:8080/api/v1/indexes/books/docs?id=book-1" \
  -H "Content-Type: application/json" \
  -d '{
    "title": "Designing Data-Intensive Applications",
    "author": "Martin Kleppmann",
    "category": "distributed-systems",
    "price": 45.00,
    "tags": ["raft", "consensus", "storage"],
    "published_year": 2017
  }'
```

**Response (`201 Created`):**
```json
{
  "id": "book-1",
  "index": "books",
  "revision": 1,
  "created_at": "2026-10-05T12:00:00Z",
  "updated_at": "2026-10-05T12:00:00Z"
}
```

---

### 2. Fetch Document by ID
`GET /api/v1/indexes/{index}/docs/{id}`

Instant zero-hop in-memory read from the local node.

```bash
curl "http://127.0.0.1:8080/api/v1/indexes/books/docs/book-1"
```

**Response (`200 OK`):**
```json
{
  "id": "book-1",
  "index": "books",
  "revision": 1,
  "data": {
    "title": "Designing Data-Intensive Applications",
    "author": "Martin Kleppmann",
    "category": "distributed-systems",
    "price": 45,
    "tags": ["raft", "consensus", "storage"],
    "published_year": 2017
  },
  "created_at": "2026-10-05T12:00:00Z",
  "updated_at": "2026-10-05T12:00:00Z"
}
```

---

### 3. Search Queries
`POST /api/v1/indexes/{index}/search`

Search queries execute against the local in-memory Bleve inverted index with zero network hops and zero CDC lag. Setting `"load_docs": true` hydrates each search hit with the full stored document.

#### Full-Text Match Query
```bash
curl -X POST "http://127.0.0.1:8080/api/v1/indexes/books/search" \
  -H "Content-Type: application/json" \
  -d '{
    "query": {
      "type": "match",
      "field": "title",
      "value": "data intensive"
    },
    "load_docs": true,
    "size": 10
  }'
```

#### Boolean & Numeric Range Query with Facets
```bash
curl -X POST "http://127.0.0.1:8080/api/v1/indexes/books/search" \
  -H "Content-Type: application/json" \
  -d '{
    "query": {
      "type": "boolean",
      "must": [
        {"type": "term", "field": "category", "value": "distributed-systems"},
        {"type": "numeric_range", "field": "price", "min": 20.0, "max": 100.0}
      ]
    },
    "facets": {
      "categories": {
        "field": "category",
        "size": 5
      },
      "price_ranges": {
        "field": "price",
        "numeric_ranges": [
          {"name": "budget", "max": 30.0},
          {"name": "premium", "min": 30.0}
        ]
      }
    },
    "load_docs": true
  }'
```

**Response (`200 OK`):**
```json
{
  "total": 1,
  "took_ns": 45000,
  "took": "45µs",
  "max_score": 1.414,
  "hits": [
    {
      "id": "book-1",
      "index": "books",
      "score": 1.414,
      "document": {
        "id": "book-1",
        "index": "books",
        "revision": 1,
        "data": {
          "title": "Designing Data-Intensive Applications",
          "author": "Martin Kleppmann",
          "category": "distributed-systems",
          "price": 45
        }
      }
    }
  ],
  "facets": {
    "categories": {
      "field": "category",
      "total": 1,
      "terms": [
        {"term": "distributed-systems", "count": 1}
      ]
    },
    "price_ranges": {
      "field": "price",
      "total": 1,
      "ranges": [
        {"name": "premium", "count": 1, "min": 30}
      ]
    }
  }
}
```

---

### 4. Atomic Batch Mutations
`POST /api/v1/indexes/{index}/batch`

Atomically index and/or delete multiple documents in a single Raft commit.

```bash
curl -X POST "http://127.0.0.1:8080/api/v1/indexes/books/batch" \
  -H "Content-Type: application/json" \
  -d '{
    "indexes": [
      {
        "id": "book-2",
        "data": {"title": "Database Internals", "author": "Alex Petrov", "price": 40.0}
      },
      {
        "id": "book-3",
        "data": {"title": "Site Reliability Engineering", "author": "Google", "price": 50.0}
      }
    ],
    "deletes": [
      {"id": "old-book-99"}
    ]
  }'
```

---

### 5. Delete a Document
`DELETE /api/v1/indexes/{index}/docs/{id}`

Atomically removes the document from both the Document Store and the Bleve index.

```bash
curl -X DELETE "http://127.0.0.1:8080/api/v1/indexes/books/docs/book-1"
```

**Response (`200 OK`):**
```json
{
  "index": "books",
  "id": "book-1",
  "deleted": true
}
```

---

### 6. Cluster Status & Health
`GET /api/v1/cluster/status`

```bash
curl "http://127.0.0.1:8080/api/v1/cluster/status"
```

**Response (`200 OK`):**
```json
{
  "node_id": "node-1",
  "is_leader": true,
  "leader_id": "node-1",
  "leader_address": "127.0.0.1:9000",
  "raft_state": "Leader",
  "committed_index": 42,
  "applied_index": 42,
  "indexes": {
    "books": {
      "doc_count": 2
    }
  },
  "nodes": [
    {"id": "node-1", "address": "127.0.0.1:9000", "voter": true, "leader": true},
    {"id": "node-2", "address": "127.0.0.1:9001", "voter": true, "leader": false},
    {"id": "node-3", "address": "127.0.0.1:9002", "voter": true, "leader": false}
  ],
  "uptime": "2h14m30s"
}
```

---

## Test Suite

Plexus-Search includes exhaustive unit tests and multi-node Raft replication tests:

```bash
# Run all tests cleanly
go test -v -count=1 ./...
```

Test coverage includes:
- **`pkg/docstore`**: Document CRUD, monotonic revisioning, batch mutations, snapshotting, and restoring.
- **`pkg/indexstore`**: Full-text Match, MatchPhrase, Term, Prefix, Fuzzy, NumericRange, Boolean clauses, Faceting, Pagination, and in-memory index rebuild from snapshot.
- **`pkg/searcher`**: Atomic cross-store coordination, local read verification, zero CDC lag assertion.
- **`pkg/api`**: All HTTP endpoints, JSON envelope parsing, 404s, batch operations, and status responses.
- **`test/`**: Full 3-node cluster replication, follower local zero-hop search validation, and distributed deletion propagation.

---

## License

Apache 2.0 / MIT
