#!/usr/bin/env python3
"""CPU profile → горизонтальные столбцы по cumulative time (pprof -top -cum). Без Graphviz."""
from __future__ import annotations

import subprocess
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
FIG = ROOT / "figures"
PROF = ROOT / "profiles" / "hashtable_large_cpu.prof"
OUT = FIG / "pprof_hashtable_cpu_top.png"


def parse_seconds(s: str) -> float | None:
    s = s.strip()
    if s == "0":
        return 0.0
    if s.endswith("s"):
        try:
            return float(s[:-1])
        except ValueError:
            return None
    return None


def main():
    if not PROF.is_file():
        print("Нет профиля:", PROF, file=sys.stderr)
        print("Запусти: go run ./cmd/profile -only=hashtable", file=sys.stderr)
        sys.exit(1)
    try:
        import matplotlib.pyplot as plt
    except ImportError:
        print("Нужен matplotlib (как для charts-mpl)", file=sys.stderr)
        sys.exit(1)

    p = subprocess.run(
        ["go", "tool", "pprof", "-top", "-cum", str(PROF)],
        cwd=ROOT,
        capture_output=True,
        text=True,
    )
    if p.returncode != 0:
        print(p.stderr, file=sys.stderr)
        sys.exit(p.returncode)
    text = p.stdout

    best: dict[str, float] = {}
    for line in text.splitlines():
        line = line.strip()
        if not line or line.startswith("File:") or line.startswith("Type:"):
            continue
        if line.startswith("Duration:") or line.startswith("Showing nodes") or line.startswith("Dropped"):
            continue
        if "flat" in line and "flat%" in line and "cum%" in line:
            continue
        parts = line.split()
        if len(parts) < 6:
            continue
        cum_s = parts[3]
        name = " ".join(parts[5:])
        cum = parse_seconds(cum_s)
        if cum is None or cum < 0.15:
            continue
        if name not in best or cum > best[name]:
            best[name] = cum

    items = sorted(best.items(), key=lambda t: -t[1])[:14]
    if not items:
        print("Не удалось разобрать pprof -top", file=sys.stderr)
        sys.exit(1)

    labels = []
    vals = []
    for name, cum in reversed(items):
        short = name if len(name) <= 54 else name[:51] + "..."
        labels.append(short)
        vals.append(cum)

    FIG.mkdir(parents=True, exist_ok=True)
    fig, ax = plt.subplots(figsize=(11, 6), dpi=120)
    ax.barh(labels, vals, color="#2563eb")
    ax.set_xlabel("cumulative CPU time (сек., сэмплы pprof)")
    ax.set_title("CPU profile: hashtable large (pprof -top -cum)")
    ax.grid(True, axis="x", alpha=0.3)
    fig.tight_layout()
    fig.savefig(OUT)
    plt.close(fig)
    print("OK", OUT)


if __name__ == "__main__":
    main()
