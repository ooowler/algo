#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
PROF="${1:-$ROOT/profiles/hashtable_large_cpu.prof}"
OUT="${2:-$ROOT/figures/pprof_hashtable_flamegraph.html}"
PORT="${PPROF_PORT:-17350}"

if [[ ! -f "$PROF" ]]; then
  echo "Нет файла профиля: $PROF" >&2
  echo "Сначала: go run ./cmd/profile -only=hashtable" >&2
  exit 1
fi

if ! command -v dot >/dev/null 2>&1; then
  echo "Для flame graph в go tool pprof нужен Graphviz (команда dot)." >&2
  echo "macOS: brew install graphviz   Debian/Ubuntu: sudo apt install graphviz" >&2
  exit 1
fi

go tool pprof -http="127.0.0.1:$PORT" -no_browser "$PROF" &
PP=$!
sleep 2
curl -fsSL "http://127.0.0.1:$PORT/flamegraph" -o "$OUT"
kill "$PP" 2>/dev/null || true
wait "$PP" 2>/dev/null || true
echo "OK $OUT (открой в браузере; в README можно дать ссылку на файл)"
