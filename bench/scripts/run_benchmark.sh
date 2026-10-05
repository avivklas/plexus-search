#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
BENCH_DIR="$REPO_ROOT/bench"
TRACK_DIR="$BENCH_DIR/track"
RESULTS_DIR="$REPO_ROOT/results"

mkdir -p "$RESULTS_DIR"

echo "=========================================================="
echo " Plexus-Search vs. Elasticsearch Rally Benchmark Suite"
echo "=========================================================="

# 1. Ensure Track is generated
if [ ! -f "$TRACK_DIR/track.json" ] || [ ! -f "$TRACK_DIR/documents.json.bz2" ]; then
    echo "[-] Generating benchmark track..."
    python3 "$BENCH_DIR/scripts/generate_track.py"
fi

# 2. Check Elasticsearch
echo "[*] Checking Elasticsearch on 127.0.0.1:9200..."
if ! curl -s "http://127.0.0.1:9200/" > /dev/null; then
    echo "[+] Starting Elasticsearch Docker container..."
    docker run -d --name es-benchmark -p 9200:9200 \
        -e "discovery.type=single-node" \
        -e "xpack.security.enabled=false" \
        -e "ES_JAVA_OPTS=-Xms1g -Xmx1g" \
        elastic/elasticsearch:8.14.0 || docker start es-benchmark
    echo "[+] Waiting for Elasticsearch to become ready..."
    for i in {1..30}; do
        if curl -s "http://127.0.0.1:9200/" > /dev/null; then
            break
        fi
        sleep 1
    done
fi
echo "[✓] Elasticsearch is ready."

# 3. Build & Start Plexus-Search
echo "[*] Building plexus-search..."
(cd "$REPO_ROOT" && go build -o bin/plexus-search ./cmd/plexus-search)

echo "[*] Starting Plexus-Search on 127.0.0.1:8080..."
DATA_DIR="/tmp/plexus_bench_data_$(date +%s)"
rm -rf "$DATA_DIR"
"$REPO_ROOT/bin/plexus-search" \
    --id node-1 \
    --http-addr 127.0.0.1:8080 \
    --raft-addr 127.0.0.1:9090 \
    --data-dir "$DATA_DIR" \
    --bootstrap > "$RESULTS_DIR/plexus_server.log" 2>&1 &
PLEXUS_PID=$!

cleanup() {
    echo "[*] Cleaning up background processes..."
    kill "$PLEXUS_PID" 2>/dev/null || true
    rm -rf "$DATA_DIR" 2>/dev/null || true
}
trap cleanup EXIT

echo "[+] Waiting for Plexus-Search HTTP API..."
for i in {1..20}; do
    if curl -s "http://127.0.0.1:8080/_cluster/health" > /dev/null; then
        break
    fi
    sleep 0.5
done
echo "[✓] Plexus-Search is ready."

ES_RACE_ID="es-benchmark-$(date +%s)"
PLEXUS_RACE_ID="plexus-benchmark-$(date +%s)"

# 4. Benchmark Elasticsearch with Rally
echo "=========================================================="
echo " [1/3] Benchmarking Elasticsearch (Target: 127.0.0.1:9200)"
echo " Race ID: $ES_RACE_ID"
echo "=========================================================="
esrally race \
    --kill-running-processes \
    --pipeline=benchmark-only \
    --target-hosts=http://127.0.0.1:9200 \
    --track-path="$TRACK_DIR" \
    --race-id="$ES_RACE_ID" \
    --report-format=markdown \
    --report-file="$RESULTS_DIR/elasticsearch_results.md"

# 5. Benchmark Plexus-Search with Rally
echo "=========================================================="
echo " [2/3] Benchmarking Plexus-Search (Target: 127.0.0.1:8080)"
echo " Race ID: $PLEXUS_RACE_ID"
echo "=========================================================="
esrally race \
    --kill-running-processes \
    --pipeline=benchmark-only \
    --target-hosts=http://127.0.0.1:8080 \
    --track-path="$TRACK_DIR" \
    --race-id="$PLEXUS_RACE_ID" \
    --report-format=markdown \
    --report-file="$RESULTS_DIR/plexus_results.md"

# 6. Compare Baseline (Elasticsearch) vs Contender (Plexus-Search)
echo "=========================================================="
echo " [3/3] Generating Official Rally Comparison Report"
echo "=========================================================="
esrally compare \
    --baseline="$ES_RACE_ID" \
    --contender="$PLEXUS_RACE_ID" \
    --report-format=markdown \
    --report-file="$RESULTS_DIR/rally_comparison.md"

cat "$RESULTS_DIR/rally_comparison.md"

echo "=========================================================="
echo "[✓] Benchmark completed successfully!"
echo "    - Elasticsearch: $RESULTS_DIR/elasticsearch_results.md"
echo "    - Plexus-Search: $RESULTS_DIR/plexus_results.md"
echo "    - Comparison:    $RESULTS_DIR/rally_comparison.md"
echo "=========================================================="
