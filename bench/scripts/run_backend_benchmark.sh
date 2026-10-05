#!/usr/bin/env bash
# Benchmarks Plexus-Search once per document storage backend (mem, pebble, s3/MinIO)
# using the same Rally track, then compares each backend against the in-memory baseline.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
TRACK_DIR="$REPO_ROOT/bench/track"
RESULTS_DIR="$REPO_ROOT/results/backends"
BACKENDS="${BACKENDS:-mem pebble s3}"
MINIO_PORT="${MINIO_PORT:-9100}"
MINIO_USER=minioadmin
MINIO_PASS=minioadmin
BUCKET=plexus-docs
STAMP="$(date +%s)"

mkdir -p "$RESULTS_DIR"

if [ ! -f "$TRACK_DIR/track.json" ] || [ ! -f "$TRACK_DIR/documents.json.bz2" ]; then
    python3 "$REPO_ROOT/bench/scripts/generate_track.py"
fi

echo "[*] Building plexus-search..."
(cd "$REPO_ROOT" && go build -o bin/plexus-search ./cmd/plexus-search)

PLEXUS_PID=""
DATA_DIR=""
stop_plexus() {
    if [ -n "$PLEXUS_PID" ]; then
        kill "$PLEXUS_PID" 2>/dev/null || true
        wait "$PLEXUS_PID" 2>/dev/null || true
        PLEXUS_PID=""
    fi
    [ -n "$DATA_DIR" ] && rm -rf "$DATA_DIR"
    DATA_DIR=""
}
MINIO_PID=""
MINIO_DIR=""
stop_minio() {
    if [ -n "$MINIO_PID" ]; then
        kill "$MINIO_PID" 2>/dev/null || true
        wait "$MINIO_PID" 2>/dev/null || true
        MINIO_PID=""
    fi
    [ -n "$MINIO_DIR" ] && rm -rf "$MINIO_DIR"
    MINIO_DIR=""
}
cleanup() {
    stop_plexus
    stop_minio
}
trap cleanup EXIT

# MinIO no longer publishes official Docker images, so run a native binary.
# Install with: go install github.com/minio/minio@latest   (or set MINIO_BIN)
MINIO_BIN="${MINIO_BIN:-$(command -v minio || echo "$REPO_ROOT/bin/minio")}"

start_minio() {
    MINIO_DIR="/tmp/plexus_bench_minio_$STAMP"
    rm -rf "$MINIO_DIR"
    MINIO_ROOT_USER="$MINIO_USER" MINIO_ROOT_PASSWORD="$MINIO_PASS" \
        "$MINIO_BIN" server "$MINIO_DIR" --address "127.0.0.1:$MINIO_PORT" --console-address "127.0.0.1:0" \
        > "$RESULTS_DIR/server_minio.log" 2>&1 &
    MINIO_PID=$!
    for _ in {1..30}; do
        curl -s "http://127.0.0.1:$MINIO_PORT/minio/health/ready" >/dev/null && break
        sleep 1
    done
}

RACE_IDS=()
for backend in $BACKENDS; do
    echo "=========================================================="
    echo " Backend: $backend"
    echo "=========================================================="
    DATA_DIR="/tmp/plexus_bench_${backend}_$STAMP"
    rm -rf "$DATA_DIR"
    extra=(--doc-store "$backend")
    if [ "$backend" = "s3" ]; then
        start_minio
        export AWS_ACCESS_KEY_ID="$MINIO_USER" AWS_SECRET_ACCESS_KEY="$MINIO_PASS" AWS_REGION=us-east-1
        extra+=(--doc-s3-bucket "$BUCKET" --doc-s3-endpoint "http://127.0.0.1:$MINIO_PORT" --doc-s3-path-style)
    fi

    "$REPO_ROOT/bin/plexus-search" --id node-1 --http-addr 127.0.0.1:8080 \
        --raft-addr 127.0.0.1:9090 --data-dir "$DATA_DIR" --bootstrap "${extra[@]}" \
        > "$RESULTS_DIR/server_$backend.log" 2>&1 &
    PLEXUS_PID=$!
    for _ in {1..40}; do
        curl -s "http://127.0.0.1:8080/_cluster/health" >/dev/null && break
        sleep 0.5
    done

    race_id="plexus-$backend-$STAMP"
    esrally race --kill-running-processes --pipeline=benchmark-only \
        --target-hosts=http://127.0.0.1:8080 --track-path="$TRACK_DIR" \
        --race-id="$race_id" --report-format=markdown \
        --report-file="$RESULTS_DIR/$backend.md"
    RACE_IDS+=("$race_id")

    stop_plexus
    [ "$backend" = "s3" ] && stop_minio || true
done

# Compare every backend against the first one (baseline)
baseline="${RACE_IDS[0]}"
for i in "${!RACE_IDS[@]}"; do
    [ "$i" -eq 0 ] && continue
    name="$(echo "$BACKENDS" | cut -d' ' -f$((i + 1)))"
    esrally compare --baseline="$baseline" --contender="${RACE_IDS[$i]}" \
        --report-format=markdown --report-file="$RESULTS_DIR/compare_mem_vs_$name.md"
done
echo "[✓] Results in $RESULTS_DIR"
