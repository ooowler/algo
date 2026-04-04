#!/usr/bin/env python3
"""Несколько прогонов go test bench → среднее, σ → таблица + график с «облаком» (fill_between)."""
from __future__ import annotations

import os
import re
import statistics
import subprocess
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
FIG = ROOT / "figures"
RESULTS = ROOT / "results"

LINE_RE = re.compile(
    r"^BenchmarkGrowthDisk_(Set|Get)/N(\d+)-\d+\s+\d+\s+(\d+(?:\.\d+)?)\s+ns/op\s+(\d+)\s+B/op"
)
RUNS = int(os.environ.get("BENCH_RUNS", "5"))
BENCHTIME = os.environ.get("BENCHTIME", "250ms")


def run_once(which: str) -> str:
    cmd = [
        "go",
        "test",
        "./hashtable",
        "-run=^$",
        f"-bench=BenchmarkGrowthDisk_{which}",
        "-benchmem",
        f"-benchtime={BENCHTIME}",
        "-count=1",
        "-timeout=120s",
    ]
    p = subprocess.run(cmd, cwd=ROOT, capture_output=True, text=True)
    if p.returncode != 0:
        print(p.stdout + p.stderr, file=sys.stderr)
        sys.exit(p.returncode)
    return p.stdout + p.stderr


def parse_block(text: str) -> dict[int, tuple[float, int]]:
    m: dict[int, tuple[float, int]] = {}
    for line in text.splitlines():
        line = line.strip()
        mo = LINE_RE.match(line)
        if not mo:
            continue
        n = int(mo.group(2))
        m[n] = (float(mo.group(3)), int(mo.group(4)))
    return m


def collect(which: str) -> tuple[dict[int, list[float]], dict[int, int]]:
    by_n: dict[int, list[float]] = {}
    last_b: dict[int, int] = {}
    for _ in range(RUNS):
        raw = run_once(which)
        snap = parse_block(raw)
        for n, (nsop, b) in snap.items():
            by_n.setdefault(n, []).append(nsop)
            last_b[n] = b
    return by_n, last_b


def main():
    try:
        import matplotlib.pyplot as plt
    except ImportError:
        print("Нужен matplotlib", file=sys.stderr)
        sys.exit(1)

    RESULTS.mkdir(parents=True, exist_ok=True)
    FIG.mkdir(parents=True, exist_ok=True)
    md: list[str] = []

    for key in ("Set", "Get"):
        data, last_b = collect(key)
        if not data:
            print("Нет данных для", key, file=sys.stderr)
            sys.exit(1)
        ns = sorted(data.keys())
        md.append(f"\n### Таблица: дисковая таблица, {key} (среднее ± σ нс/op, {RUNS} прогонов по {BENCHTIME})\n\n")
        md.append("| N | нс/op | B/op (последний прогон) | примерно op/с |\n")
        md.append("|---|-------|-------------------------|---------------|\n")
        arr_ns = list(ns)
        arr_m = []
        arr_s = []
        for n in arr_ns:
            vals = data[n]
            mu = statistics.mean(vals)
            sd = statistics.stdev(vals) if len(vals) > 1 else 0.0
            arr_m.append(mu)
            arr_s.append(sd)
            thr = 1e9 / mu if mu else 0
            md.append(f"| {n} | {mu:,.1f} ± {sd:,.1f} | {last_b[n]} | {thr:,.0f} |\n")

        fig, ax = plt.subplots(figsize=(9, 5), dpi=120)
        ax.plot(arr_ns, arr_m, "o-", color="#2563eb", linewidth=2, markersize=7, label="среднее")
        ax.errorbar(arr_ns, arr_m, yerr=arr_s, fmt="none", ecolor="#1e3a5f", capsize=4, alpha=0.9)
        lo = [a - b for a, b in zip(arr_m, arr_s)]
        hi = [a + b for a, b in zip(arr_m, arr_s)]
        ax.fill_between(arr_ns, lo, hi, alpha=0.25, color="#38bdf8", label="±1σ")
        ax.set_xlabel("N")
        ax.set_ylabel("Latency, ns/op")
        ax.set_title(f"Дисковая таблица — BenchmarkGrowthDisk_{key}")
        ax.set_xscale("log")
        ax.grid(True, alpha=0.3)
        ax.legend(loc="upper left")
        fig.tight_layout()
        path = FIG / f"disk_{key.lower()}_latency_cloud.png"
        fig.savefig(path)
        plt.close(fig)
        print("OK", path)

    (RESULTS / "disk_bench_table.md").write_text("".join(md), encoding="utf-8")
    print("OK", RESULTS / "disk_bench_table.md")


if __name__ == "__main__":
    main()
