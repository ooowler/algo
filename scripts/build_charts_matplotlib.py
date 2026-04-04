#!/usr/bin/env python3
"""go test BenchmarkGrowth (несколько прогонов) → parse → matplotlib с ±σ (облако)."""
from __future__ import annotations

import os
import re
import statistics
import subprocess
import sys
from collections import defaultdict
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
FIG = ROOT / "figures"
RESULTS = ROOT / "results"

LINE_RE = re.compile(
    r"^(BenchmarkGrowth\S+)/N(\d+)-\d+\s+\d+\s+(\d+(?:\.\d+)?)\s+ns/op\s+(\d+)\s+B/op"
)

BENCH_COUNT = int(os.environ.get("GROWTH_BENCH_COUNT", "3"))
BENCHTIME = os.environ.get("GROWTH_BENCHTIME", "150ms")


def run_bench() -> str:
    RESULTS.mkdir(parents=True, exist_ok=True)
    timeout = "600s" if BENCH_COUNT > 1 else "180s"
    cmd = [
        "go",
        "test",
        "./hashtable",
        "./perfecthash",
        "./lsh",
        "-run=^$",
        "-bench=BenchmarkGrowth",
        "-benchmem",
        f"-benchtime={BENCHTIME}",
        f"-count={BENCH_COUNT}",
        f"-timeout={timeout}",
    ]
    p = subprocess.run(cmd, cwd=ROOT, capture_output=True, text=True)
    out = p.stdout + p.stderr
    (RESULTS / "growth_bench.txt").write_text(out, encoding="utf-8")
    if p.returncode != 0:
        print(out, file=sys.stderr)
        sys.exit(p.returncode)
    return out


def parse_samples(text: str) -> dict[str, dict[int, list[tuple[float, float]]]]:
    m: dict[str, dict[int, list[tuple[float, float]]]] = defaultdict(lambda: defaultdict(list))
    for line in text.splitlines():
        line = line.strip()
        mo = LINE_RE.match(line)
        if not mo:
            continue
        name, n_s, ns_s, b_s = mo.groups()
        m[name][int(n_s)].append((float(ns_s), float(b_s)))
    return {k: dict(v) for k, v in m.items()}


def agg(by_n: dict[int, list[tuple[float, float]]]) -> list[tuple[int, float, float, float, float]]:
    rows: list[tuple[int, float, float, float, float]] = []
    for n in sorted(by_n.keys()):
        samples = by_n[n]
        nsv = [s[0] for s in samples]
        bv = [s[1] for s in samples]
        mn = statistics.mean(nsv)
        sn = statistics.stdev(nsv) if len(nsv) > 1 else 0.0
        mb = statistics.mean(bv)
        sb = statistics.stdev(bv) if len(bv) > 1 else 0.0
        rows.append((n, mn, sn, mb, sb))
    return rows


def plot_xy_cloud(
    rows: list[tuple[int, float, float, float, float]],
    y_from_row,
    title: str,
    xlab: str,
    ylab: str,
    path: Path,
    logx: bool = True,
    logy: bool = False,
):
    import matplotlib.pyplot as plt

    if not rows:
        return
    xs = [r[0] for r in rows]
    ym = [y_from_row(r)[0] for r in rows]
    ys = [y_from_row(r)[1] for r in rows]
    fig, ax = plt.subplots(figsize=(10, 5), dpi=120)
    ax.plot(xs, ym, "o-", linewidth=2, markersize=6, color="#2563eb", label="среднее")
    if logy:
        lo = [max(a - b, 1e-15) for a, b in zip(ym, ys)]
        hi = [a + b for a, b in zip(ym, ys)]
    else:
        lo = [max(a - b, 0.0) for a, b in zip(ym, ys)]
        hi = [a + b for a, b in zip(ym, ys)]
    ax.fill_between(xs, lo, hi, alpha=0.25, color="#38bdf8", label="±1σ")
    ax.errorbar(xs, ym, yerr=ys, fmt="none", ecolor="#1e3a5f", capsize=4, alpha=0.85)
    ax.set_title(title + (f" ({BENCH_COUNT} прогона)" if BENCH_COUNT > 1 else ""))
    ax.set_xlabel(xlab)
    ax.set_ylabel(ylab)
    if logx:
        ax.set_xscale("log")
    if logy:
        ax.set_yscale("log")
    ax.grid(True, alpha=0.3)
    ax.legend(loc="best", fontsize=8)
    fig.tight_layout()
    fig.savefig(path)
    plt.close(fig)


def plot_dual_cloud(
    rows_a: list[tuple[int, float, float, float, float]],
    rows_b: list[tuple[int, float, float, float, float]],
    title: str,
    xlab: str,
    ylab: str,
    path: Path,
    label_a: str,
    label_b: str,
):
    import matplotlib.pyplot as plt

    fig, ax = plt.subplots(figsize=(10, 5), dpi=120)

    def scale_row(r):
        m = r[1] / 1e6
        s = r[2] / 1e6
        return m, s

    for rows, col, lab, mk in (
        (rows_a, "#2563eb", label_a, "o"),
        (rows_b, "#d97706", label_b, "s"),
    ):
        if not rows:
            continue
        xs = [r[0] for r in rows]
        ym = [scale_row(r)[0] for r in rows]
        ys = [scale_row(r)[1] for r in rows]
        ax.plot(xs, ym, mk + "-", linewidth=2, color=col, label=lab)
        lo = [max(m - s, 1e-9) for m, s in zip(ym, ys)]
        hi = [m + s for m, s in zip(ym, ys)]
        ax.fill_between(xs, lo, hi, alpha=0.2, color=col)
        ax.errorbar(xs, ym, yerr=ys, fmt="none", ecolor=col, capsize=3, alpha=0.7)
    ax.set_title(title + (f" ({BENCH_COUNT} прогона)" if BENCH_COUNT > 1 else ""))
    ax.set_xlabel(xlab)
    ax.set_ylabel(ylab)
    ax.set_xscale("log")
    ax.set_yscale("log")
    ax.legend()
    ax.grid(True, alpha=0.3)
    fig.tight_layout()
    fig.savefig(path)
    plt.close(fig)


def plot_multi_cloud(
    groups: list[tuple[str, list[tuple[int, float, float, float, float]]]],
    title: str,
    xlab: str,
    ylab: str,
    path: Path,
):
    import matplotlib.pyplot as plt

    fig, ax = plt.subplots(figsize=(10, 5), dpi=120)
    colors = ["#2563eb", "#059669", "#d97706"]
    for i, (label, rows) in enumerate(groups):
        if not rows:
            continue
        col = colors[i % len(colors)]
        xs = [r[0] for r in rows]
        ym = [r[1] / 1e3 for r in rows]
        ys = [r[2] / 1e3 for r in rows]
        ax.plot(xs, ym, "o-", linewidth=2, color=col, label=label)
        lo = [max(a - b, 1e-9) for a, b in zip(ym, ys)]
        hi = [a + b for a, b in zip(ym, ys)]
        ax.fill_between(xs, lo, hi, alpha=0.2, color=col)
        ax.errorbar(xs, ym, yerr=ys, fmt="none", ecolor=col, capsize=3, alpha=0.7)
    ax.set_title(title + (f" ({BENCH_COUNT} прогона)" if BENCH_COUNT > 1 else ""))
    ax.set_xlabel(xlab)
    ax.set_ylabel(ylab)
    ax.set_xscale("log")
    ax.set_yscale("log")
    ax.legend()
    ax.grid(True, alpha=0.3)
    fig.tight_layout()
    fig.savefig(path)
    plt.close(fig)


def main():
    try:
        import matplotlib  # noqa: F401
    except ImportError:
        print("Установи matplotlib:", file=sys.stderr)
        print("  python3 -m venv .venv && .venv/bin/pip install -r scripts/requirements.txt", file=sys.stderr)
        sys.exit(1)

    FIG.mkdir(parents=True, exist_ok=True)
    raw = run_bench()
    samples = parse_samples(raw)

    def take(name: str):
        return agg(samples.get(name, {}))

    ph_b = take("BenchmarkGrowthPH_Build")
    ph_g = take("BenchmarkGrowthPH_Get")
    lsh = take("BenchmarkGrowthLSH_Find")
    naive = take("BenchmarkGrowthLSH_Naive")
    d_set = take("BenchmarkGrowthDisk_Set")
    d_get = take("BenchmarkGrowthDisk_Get")

    if ph_b:
        plot_xy_cloud(
            ph_b,
            lambda r: (r[1] / 1e6, r[2] / 1e6),
            "Perfect hash: Build — время",
            "N, ключей",
            "мс/op",
            FIG / "growth_ph_build_time.png",
        )
        plot_xy_cloud(
            ph_b,
            lambda r: (r[3] / 1024, r[4] / 1024),
            "Perfect hash: Build — память",
            "N, ключей",
            "KiB/op",
            FIG / "growth_ph_build_mem.png",
        )

    if ph_g:
        plot_xy_cloud(
            ph_g,
            lambda r: (r[1], r[2]),
            "Perfect hash: Get — время",
            "N, ключей",
            "нс/op",
            FIG / "growth_ph_get_time.png",
            logy=False,
        )
        plot_xy_cloud(
            ph_g,
            lambda r: (r[3] / 1024, r[4] / 1024),
            "Perfect hash: Get — память",
            "N, ключей",
            "KiB/op",
            FIG / "growth_ph_get_mem.png",
        )

    if lsh:
        plot_xy_cloud(
            lsh,
            lambda r: (r[1] / 1e6, r[2] / 1e6),
            "LSH FindDuplicates — время",
            "N, точек",
            "мс/op",
            FIG / "growth_lsh_find_time.png",
        )
        plot_xy_cloud(
            lsh,
            lambda r: (r[3] / 1024, r[4] / 1024),
            "LSH FindDuplicates — память",
            "N, точек",
            "KiB/op",
            FIG / "growth_lsh_find_mem.png",
        )

    if naive:
        plot_xy_cloud(
            naive,
            lambda r: (r[1] / 1e6, r[2] / 1e6),
            "Наивный скан — время",
            "N, точек",
            "мс/op",
            FIG / "growth_naive_time.png",
        )
        plot_xy_cloud(
            naive,
            lambda r: (r[3] / 1024, r[4] / 1024),
            "Наивный скан — память",
            "N, точек",
            "KiB/op",
            FIG / "growth_naive_mem.png",
        )

    plot_dual_cloud(
        lsh,
        naive,
        "LSH vs наивный (время)",
        "N, точек",
        "мс/op",
        FIG / "growth_dup_compare_time.png",
        "LSH",
        "Naive",
    )

    plot_multi_cloud(
        [
            ("Set", d_set),
            ("Get", d_get),
        ],
        "Дисковая таблица: Set и Get",
        "N",
        "мкс/op",
        FIG / "growth_disk_time.png",
    )

    print(f"OK: figures/growth_*.png (прогонов: {BENCH_COUNT}, benchtime={BENCHTIME})")


if __name__ == "__main__":
    main()
