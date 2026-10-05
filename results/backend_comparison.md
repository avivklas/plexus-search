# Benchmark Results: Plexus-Search Docstore Backends

This benchmark evaluates **Plexus-Search** across its three document storage backends (**In-Memory**, **Pebble LSM**, and **S3 / MinIO**) using the official **Elasticsearch Rally** (`esrally 2.13.0`) benchmarking suite.

The inverted search index is kept strictly in-memory (Bleve) across all configurations, while raw JSON document storage is delegated to the evaluated backend.

---

## 1. Executive Summary & Comparison Matrix

All tests were executed on the same macOS host under identical concurrency (2 clients) and track specifications (5,000 multi-field technical documents, bulk batch size of 250 docs):

| Metric / Workload | In-Memory (Baseline) | Pebble LSM (Embedded) | S3 / MinIO (Object Store) | Delta (Mem vs Pebble) | Delta (Mem vs S3) |
|---|:---:|:---:|:---:|:---:|:---:|
| **Bulk Ingest Throughput** | **2,347.46 docs/s** | 2,206.83 docs/s | 876.84 docs/s | -5.99% | **-62.65%** |
| **Bulk Ingest p50 Latency** | **118.60 ms** | 119.94 ms | 261.16 ms | +1.14% | **+120.21%** |
| **Bulk Ingest p90 Latency** | **135.73 ms** | 135.14 ms | 359.05 ms | -0.44% | **+164.54%** |
| **Term Filter (`term-category`)** | 7,891.64 ops/s (**0.13 ms**) | 6,814.19 ops/s (**0.14 ms**) | **8,829.71 ops/s** (**0.11 ms**) | -13.65% | **+11.89%** |
| **Compound Boolean (`bool-compound`)** | 7,093.30 ops/s (**0.14 ms**) | **7,402.76 ops/s** (**0.13 ms**) | 6,683.81 ops/s (**0.14 ms**) | +4.36% | -5.77% |
| **Analyzed Search (`match-title`)** | **2,894.66 ops/s** (**0.40 ms**) | 2,684.70 ops/s (**0.40 ms**) | 453.94 ops/s (**3.06 ms**) | -7.25% | **-84.32%** |
| **Match Phrase (`match-phrase`)** | 279.52 ops/s (5.16 ms) | **353.31 ops/s** (**4.58 ms**) | 169.64 ops/s (7.47 ms) | **+26.40%** | -39.31% |
| **Numeric Range Query** | **120.27 ops/s** (13.74 ms) | 119.94 ops/s (14.54 ms) | 103.33 ops/s (16.52 ms) | -0.28% | -14.09% |
| **Match All Scans** | **47.28 ops/s** (42.49 ms) | 45.59 ops/s (43.71 ms) | 37.65 ops/s (47.16 ms) | -3.58% | -20.37% |

---

## 2. Key Insights & Architectural Takeaways

### 1. Pebble LSM: Near Zero Overhead with Embedded Durability
- **Bulk Ingestion Overhead is minimal (-5.99%)**: Commits are buffered atomically using Pebble's write batch and LSM memtable. Write latency p50 remains practically flat (118.6ms vs 119.9ms).
- **Search Query Latencies are virtually identical**: Because the Bleve index resides in-memory, query execution time is unchanged. Hydration of top search hits via local Pebble key lookups incurs sub-millisecond costs.
- **Production Recommendation**: Pebble is the best default choice for persistent nodes, providing instant recovery without external infrastructure dependencies.

### 2. S3 / MinIO: Scalable Cold Storage with Network Bound Throughput
- **Network Round-Trip on Ingestion (-62.65% throughput)**: Ingestion drops from ~2,347 docs/s to ~877 docs/s. Each batch of 250 documents requires parallel HTTP PUT requests to S3/MinIO. Even with a concurrency pool of 16 workers, HTTP framing and TCP roundtrips dominate the write phase.
- **Hit Hydration Impact on Full-Text Search (-84.32% on `match-title`)**: When queries return matching documents that require full document bodies, fetching the documents over S3 `GetObject` introduces a ~2.65 ms latency tax per search request.
- **Zero Impact on Term Filter Queries (+11.89% / parity)**: Queries that only inspect indexed terms or return low doc counts stay at sub-millisecond latencies (~0.11 ms) because they run directly against the in-memory inverted index.
- **Production Recommendation**: S3 is ideal for massive corpora where local disk size is constrained, or when Plexus is deployed as a stateless compute layer over durable cloud storage.

---

## 3. Raw Rally Comparison Reports

- **Pebble vs In-Memory**: [compare_mem_vs_pebble.md](results/backends/compare_mem_vs_pebble.md)
- **S3 vs In-Memory**: [compare_mem_vs_s3.md](results/backends/compare_mem_vs_s3.md)
- **Individual Backend Results**:
  - In-Memory: [mem.md](results/backends/mem.md)
  - Pebble: [pebble.md](results/backends/pebble.md)
  - S3 / MinIO: [s3.md](results/backends/s3.md)
