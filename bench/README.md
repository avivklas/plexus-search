# Plexus-Search vs. Elasticsearch Benchmark Suite

This directory contains the automated benchmarking suite comparing **Plexus-Search** against **Elasticsearch** using Elasticsearch's official benchmarking tool, **[Elasticsearch Rally](https://github.com/elastic/rally)** (`esrally`).

---

## Directory Structure

```
bench/
├── README.md                  # This documentation
├── scripts/
│   ├── generate_track.py      # Generates Rally benchmark track with dataset & queries
│   └── run_benchmark.sh       # Automated runner for Elasticsearch vs Plexus-Search
└── track/
    ├── documents.json.bz2     # Compressed corpus (5,000 realistic documents)
    ├── index.json             # Elasticsearch-compatible index mapping & settings
    └── track.json             # Rally track definition (operations, challenges, workloads)
```

---

## Benchmark Design

### Why Elasticsearch Rally?
[Rally](https://esrally.readthedocs.io/) is Elastic's standard macrobenchmarking framework used by the Elasticsearch engineering team to measure throughput, latency percentiles, JVM GC behavior, and error rates across releases.

Because Plexus-Search implements an **Elasticsearch-compatible REST API** (`/`, `/_cluster/health`, `/{index}/_search`, `/_bulk`, `/{index}/_refresh`, `/{index}/_doc/{id}`), Rally targets both engines identically using the exact same track, operations, and client connections.

### Workloads Evaluated
1. **Bulk Ingestion (`bulk-index`)**: Tests streaming batch mutations with 250 documents per bulk request.
2. **Analyzed Full-Text Search (`match-title-query`)**: Tests tokenized text matching.
3. **Exact Term Filter (`term-category-query`)**: Tests exact unanalyzed keyword lookups.
4. **Phrase Search (`match-phrase-query`)**: Tests exact term sequence positional lookups.
5. **Numeric Range (`numeric-range-query`)**: Tests range filters across numeric boundaries.
6. **Compound Boolean (`bool-compound-query`)**: Tests combined `must` + `filter` clauses.

---

## Quickstart: Running the Benchmark

### Prerequisites
1. Docker (for running the official Elasticsearch 8.14.0 image)
2. Go 1.22+ (for building Plexus-Search)
3. Python 3.10+ with `esrally` installed (`pip install esrally`)

### One-Command Automated Run
```bash
./bench/scripts/run_benchmark.sh
```

The script will automatically:
1. Generate the track dataset (5,000 documents compressed with bz2).
2. Start Elasticsearch in Docker on port 9200 (if not already running).
3. Build and launch Plexus-Search on port 8080 with embedded Raft consensus.
4. Run Rally against Elasticsearch and save metrics to `results/elasticsearch_results.md`.
5. Run Rally against Plexus-Search and save metrics to `results/plexus_results.md`.
6. Run `esrally compare` and save the official comparison report to `results/rally_comparison.md`.

---

## Results Summary

See [results/RESULTS.md](../results/RESULTS.md) for full benchmark numbers, latency percentiles, and architectural analysis.
