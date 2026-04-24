#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."
export SYNC_README="${SYNC_README:-1}"
export GROWTH_BENCH_COUNT="${GROWTH_BENCH_COUNT:-5}"
export GROWTH_BENCHTIME="${GROWTH_BENCHTIME:-200ms}"
export GROWTH_TIMEOUT="${GROWTH_TIMEOUT:-3600s}"
export GOCACHE="${GOCACHE:-$PWD/.gocache}"
go test ./hashtable ./perfecthash ./lsh -timeout 900s -count=1
python3 scripts/build_charts_matplotlib.py
