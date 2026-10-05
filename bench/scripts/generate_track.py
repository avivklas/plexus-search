#!/usr/bin/env python3
"""
Generate a standardized Elasticsearch Rally benchmark track for comparing
Plexus-Search against Elasticsearch.
"""

import os
import json
import bz2
import random

def generate_track(output_dir="bench/track", num_docs=5000):
    os.makedirs(output_dir, exist_ok=True)
    random.seed(42)

    categories = [
        "distributed-systems",
        "search-engines",
        "databases",
        "cloud-native",
        "storage-engines"
    ]

    title_prefixes = [
        "Architecting", "Designing", "Optimizing", "Benchmarking", "Scaling",
        "Deep Dive into", "Mastering", "Internals of", "Fault Tolerance in", "High Performance"
    ]
    title_subjects = [
        "Raft Consensus Protocol", "Distributed Inverted Index", "State Machine Replication",
        "Zero-CDC Streaming Topologies", "Bleve Full-Text Engine", "Broken CAP Sequential Reads",
        "Direct-I/O Segmented Log Stores", "Multi-Store Raft Coordination", "LSM-Tree Storage Engines",
        "Vector Embeddings and Lucene"
    ]

    body_sentences = [
        "Plexus embeds Raft consensus directly into the application process without external dependencies.",
        "Zero CDC lag is achieved by committing mutations atomically across both raw docstore and search indices.",
        "Local memory reads bypass the consensus network hops while guaranteeing read-your-own-writes consistency.",
        "Append-only block-aligned segment log stores maximize disk write throughput with low write amplification.",
        "Elasticsearch relies on Lucene segment flushing and background merging threads to provide eventual search visibility.",
        "Consensus quorums allow majority agreement even in the face of transient node failures and network partitions."
    ]

    tags_pool = ["raft", "consensus", "bleve", "lucene", "storage", "performance", "acid", "cdc", "distributed", "nosql"]

    docs = []
    for i in range(num_docs):
        cat = random.choice(categories)
        prefix = random.choice(title_prefixes)
        subject = random.choice(title_subjects)
        title = f"{prefix} {subject} #{i}"
        body = " ".join(random.sample(body_sentences, 3))
        views = random.randint(100, 100000)
        rating = round(random.uniform(2.5, 5.0), 2)
        year = random.randint(2018, 2026)
        tags = random.sample(tags_pool, random.randint(2, 4))

        doc = {
            "id": f"doc-{i}",
            "title": title,
            "body": body,
            "category": cat,
            "views": views,
            "rating": rating,
            "published_year": year,
            "tags": tags
        }
        docs.append(json.dumps(doc))

    raw_corpus = "\n".join(docs).encode("utf-8")
    comp_corpus = bz2.compress(raw_corpus)

    corpus_file = os.path.join(output_dir, "documents.json.bz2")
    with open(corpus_file, "wb") as f:
        f.write(comp_corpus)

    print(f"Generated corpus: {num_docs} docs, {len(raw_corpus)} bytes uncompressed, {len(comp_corpus)} bytes compressed.")

    index_spec = {
        "settings": {
            "index.number_of_shards": 1,
            "index.number_of_replicas": 0
        },
        "mappings": {
            "properties": {
                "id": {"type": "keyword"},
                "title": {"type": "text"},
                "body": {"type": "text"},
                "category": {"type": "keyword"},
                "views": {"type": "integer"},
                "rating": {"type": "float"},
                "published_year": {"type": "integer"},
                "tags": {"type": "keyword"}
            }
        }
    }
    with open(os.path.join(output_dir, "index.json"), "w") as f:
        json.dump(index_spec, f, indent=2)

    track_spec = {
        "version": 2,
        "description": "Elasticsearch Rally benchmark track comparing Plexus-Search vs Elasticsearch",
        "indices": [
            {
                "name": "bench-articles",
                "body": "index.json"
            }
        ],
        "corpora": [
            {
                "name": "articles-corpus",
                "documents": [
                    {
                        "source-file": "documents.json.bz2",
                        "document-count": num_docs,
                        "compressed-bytes": len(comp_corpus),
                        "uncompressed-bytes": len(raw_corpus),
                        "target-index": "bench-articles"
                    }
                ]
            }
        ],
        "operations": [
            {
                "name": "delete-index",
                "operation-type": "delete-index"
            },
            {
                "name": "create-index",
                "operation-type": "create-index"
            },
            {
                "name": "check-cluster-health",
                "operation-type": "cluster-health",
                "request-params": {
                    "wait_for_status": "green"
                }
            },
            {
                "name": "bulk-index",
                "operation-type": "bulk",
                "bulk-size": 250
            },
            {
                "name": "refresh",
                "operation-type": "refresh"
            },
            {
                "name": "match-all-query",
                "operation-type": "search",
                "body": {
                    "query": {
                        "match_all": {}
                    }
                }
            },
            {
                "name": "match-title-query",
                "operation-type": "search",
                "body": {
                    "query": {
                        "match": {
                            "title": "Consensus"
                        }
                    }
                }
            },
            {
                "name": "match-phrase-query",
                "operation-type": "search",
                "body": {
                    "query": {
                        "match_phrase": {
                            "body": "Zero CDC lag"
                        }
                    }
                }
            },
            {
                "name": "term-category-query",
                "operation-type": "search",
                "body": {
                    "query": {
                        "term": {
                            "category": "distributed-systems"
                        }
                    }
                }
            },
            {
                "name": "numeric-range-query",
                "operation-type": "search",
                "body": {
                    "query": {
                        "range": {
                            "views": {
                                "gte": 10000,
                                "lte": 50000
                            }
                        }
                    }
                }
            },
            {
                "name": "bool-compound-query",
                "operation-type": "search",
                "body": {
                    "query": {
                        "bool": {
                            "must": [
                                {"match": {"title": "Consensus"}}
                            ],
                            "filter": [
                                {"term": {"category": "distributed-systems"}}
                            ]
                        }
                    }
                }
            }
        ],
        "challenges": [
            {
                "name": "full-comparison-challenge",
                "description": "Comprehensive benchmark suite testing bulk indexing throughput and multiple search patterns",
                "default": True,
                "schedule": [
                    {"operation": "delete-index"},
                    {"operation": "create-index"},
                    {"operation": "check-cluster-health"},
                    {"operation": "bulk-index"},
                    {"operation": "refresh"},
                    {
                        "operation": "match-all-query",
                        "clients": 2,
                        "warmup-iterations": 50,
                        "iterations": 250
                    },
                    {
                        "operation": "match-title-query",
                        "clients": 2,
                        "warmup-iterations": 50,
                        "iterations": 250
                    },
                    {
                        "operation": "match-phrase-query",
                        "clients": 2,
                        "warmup-iterations": 50,
                        "iterations": 250
                    },
                    {
                        "operation": "term-category-query",
                        "clients": 2,
                        "warmup-iterations": 50,
                        "iterations": 250
                    },
                    {
                        "operation": "numeric-range-query",
                        "clients": 2,
                        "warmup-iterations": 50,
                        "iterations": 250
                    },
                    {
                        "operation": "bool-compound-query",
                        "clients": 2,
                        "warmup-iterations": 50,
                        "iterations": 250
                    }
                ]
            }
        ]
    }

    with open(os.path.join(output_dir, "track.json"), "w") as f:
        json.dump(track_spec, f, indent=2)

    print(f"Track written to {output_dir}/track.json")

if __name__ == "__main__":
    generate_track()
