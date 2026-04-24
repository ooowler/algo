#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
PROF="${1:-$ROOT/profiles/hashtable_cpu.prof}"
OUT="${2:-$ROOT/figures/pprof_hashtable_flamegraph.html}"
PORT="${PPROF_PORT:-17350}"
URL="http://127.0.0.1:$PORT/ui/flamegraph"
LOG="$(mktemp "${TMPDIR:-/tmp}/pprof-flame.XXXXXX.log")"
SVG_OUT="${OUT%.html}.svg"
TITLE="$(basename "$PROF")"
TITLE="${TITLE%_cpu.prof}"
TITLE="${TITLE%_mem.prof}"
TITLE="${TITLE//_/ } flamegraph"
FOCUS_PREFIX="${PPROF_FLAME_FOCUS:-}"

if [[ ! -f "$PROF" ]]; then
  echo "Нет файла профиля: $PROF" >&2
  echo "Сначала: go run -C lab1 ./cmd/profile -only=hashtable (из корня algo)" >&2
  exit 1
fi

mkdir -p "$(dirname "$OUT")"

cleanup() {
  if [[ -n "${PP:-}" ]]; then
    kill "$PP" 2>/dev/null || true
    wait "$PP" 2>/dev/null || true
  fi
  rm -f "$LOG"
}
trap cleanup EXIT

go tool pprof -http="127.0.0.1:$PORT" -no_browser "$PROF" >"$LOG" 2>&1 &
PP=$!

for _ in $(seq 1 50); do
  if curl -fsSL "$URL" -o "$OUT" 2>/dev/null; then
    if command -v python3 >/dev/null 2>&1 && [[ -f "$ROOT/scripts/render_pprof_flame_svg.py" ]]; then
      if [[ -n "$FOCUS_PREFIX" ]]; then
        python3 "$ROOT/scripts/render_pprof_flame_svg.py" "$OUT" "$SVG_OUT" "$TITLE" "$FOCUS_PREFIX" >/dev/null
      else
        python3 "$ROOT/scripts/render_pprof_flame_svg.py" "$OUT" "$SVG_OUT" "$TITLE" >/dev/null
      fi
      echo "OK $OUT"
      echo "OK $SVG_OUT"
      echo "Flamegraph HTML сохранён для live UI, SVG — для README/отчёта"
      if [[ -n "$FOCUS_PREFIX" ]]; then
        echo "Focus: $FOCUS_PREFIX"
      fi
      echo "Live UI: $URL"
      exit 0
    fi
    echo "OK $OUT"
    echo "Flamegraph HTML сохранён без Graphviz; для live UI используй $URL"
    exit 0
  fi
  sleep 0.2
done

echo "Не удалось скачать flamegraph с $URL" >&2
cat "$LOG" >&2
exit 1
