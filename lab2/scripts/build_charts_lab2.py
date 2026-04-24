#!/usr/bin/env python3
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
os.environ.setdefault("MPLCONFIGDIR", str(ROOT / ".mplconfig"))
os.environ.setdefault("XDG_CACHE_HOME", str(ROOT / ".cache"))

GROWTH_RE = re.compile(
    r"^(BenchmarkGrowth\S+)/N(\d+)-\d+\s+\d+\s+(\d+(?:\.\d+)?)\s+ns/op\s+(\d+)\s+B/op"
)
RADIUS_RE = re.compile(
    r"^BenchmarkRadius(KD|Grid|Naive)_Search/R(\d+)-\d+\s+\d+\s+(\d+(?:\.\d+)?)\s+ns/op\s+(\d+)\s+B/op"
)

BENCH_COUNT = int(os.environ.get("BENCH_COUNT", "3"))
BENCHTIME = os.environ.get("BENCHTIME", "80ms")
SIGMA = 3


def run_bench() -> str:
    RESULTS.mkdir(parents=True, exist_ok=True)
    cmd = [
        "go", "test", "./geosearch",
        "-run=^$", "-bench=.",
        "-benchmem", f"-benchtime={BENCHTIME}",
        f"-count={BENCH_COUNT}", "-timeout=300s",
    ]
    p = subprocess.run(cmd, cwd=ROOT, capture_output=True, text=True)
    out = p.stdout + p.stderr
    (RESULTS / "bench.txt").write_text(out, encoding="utf-8")
    if p.returncode != 0:
        print(out, file=sys.stderr)
        sys.exit(p.returncode)
    return out


def parse_growth(text: str) -> dict[str, dict[int, list[tuple[float, float]]]]:
    m: dict[str, dict[int, list[tuple[float, float]]]] = defaultdict(lambda: defaultdict(list))
    for line in text.splitlines():
        mo = GROWTH_RE.match(line.strip())
        if not mo:
            continue
        name, n_s, ns_s, b_s = mo.groups()
        m[name][int(n_s)].append((float(ns_s), float(b_s)))
    return {k: dict(v) for k, v in m.items()}


def parse_radius(text: str) -> dict[str, dict[int, list[tuple[float, float]]]]:
    m: dict[str, dict[int, list[tuple[float, float]]]] = defaultdict(lambda: defaultdict(list))
    for line in text.splitlines():
        mo = RADIUS_RE.match(line.strip())
        if not mo:
            continue
        algo, r_s, ns_s, b_s = mo.groups()
        m[algo][int(r_s)].append((float(ns_s), float(b_s)))
    return {k: dict(v) for k, v in m.items()}


Row = tuple[int, float, float, float, float]


def agg(by_n: dict[int, list[tuple[float, float]]]) -> list[Row]:
    rows: list[Row] = []
    for n in sorted(by_n):
        samples = by_n[n]
        nsv = [s[0] for s in samples]
        bv = [s[1] for s in samples]
        mn = statistics.mean(nsv)
        sn = statistics.stdev(nsv) if len(nsv) > 1 else 0.0
        mb = statistics.mean(bv)
        sb = statistics.stdev(bv) if len(bv) > 1 else 0.0
        rows.append((n, mn, sn, mb, sb))
    return rows


def _cloud(ax, xs, ym, ys, col, label, marker="o", logy=False):
    ax.plot(xs, ym, marker + "-", linewidth=2, markersize=6, color=col, label=label)
    if not logy:
        lo = [max(a - SIGMA * b, 0.0) for a, b in zip(ym, ys)]
        hi = [a + SIGMA * b for a, b in zip(ym, ys)]
        ax.fill_between(xs, lo, hi, alpha=0.2, color=col)
        ax.errorbar(xs, ym, yerr=[SIGMA * y for y in ys], fmt="none", ecolor=col, capsize=4, alpha=0.7)


def plot_single(rows: list[Row], y_fn, title, xlab, ylab, path, logx=True, logy=False):
    import matplotlib

    matplotlib.use("Agg")
    import matplotlib.pyplot as plt
    if not rows:
        return
    xs = [r[0] for r in rows]
    ym = [y_fn(r)[0] for r in rows]
    ys = [y_fn(r)[1] for r in rows]
    fig, ax = plt.subplots(figsize=(10, 5), dpi=120)
    _cloud(ax, xs, ym, ys, "#2563eb", "среднее", logy=logy)
    ax.set_title(f"{title} (±{SIGMA}σ, {BENCH_COUNT} прогонов)")
    ax.set_xlabel(xlab)
    ax.set_ylabel(ylab)
    if logx:
        ax.set_xscale("log")
    if logy:
        ax.set_yscale("log")
    ax.grid(True, alpha=0.3)
    ax.legend(fontsize=8)
    fig.tight_layout()
    fig.savefig(path)
    plt.close(fig)


def plot_multi(groups, title, xlab, ylab, path, logx=True, logy=False):
    import matplotlib

    matplotlib.use("Agg")
    import matplotlib.pyplot as plt
    COLORS = ["#2563eb", "#059669", "#d97706"]
    MARKERS = ["o", "s", "^"]
    fig, ax = plt.subplots(figsize=(10, 5), dpi=120)
    for i, (label, rows, y_fn) in enumerate(groups):
        if not rows:
            continue
        col = COLORS[i % len(COLORS)]
        xs = [r[0] for r in rows]
        ym = [y_fn(r)[0] for r in rows]
        ys = [y_fn(r)[1] for r in rows]
        _cloud(ax, xs, ym, ys, col, label, marker=MARKERS[i], logy=logy)
    ax.set_title(f"{title} (±{SIGMA}σ, {BENCH_COUNT} прогонов)")
    ax.set_xlabel(xlab)
    ax.set_ylabel(ylab)
    if logx:
        ax.set_xscale("log")
    if logy:
        ax.set_yscale("log")
    ax.legend()
    ax.grid(True, alpha=0.3)
    fig.tight_layout()
    fig.savefig(path)
    plt.close(fig)


def write_tables(
    path: Path,
    kd_build: list[Row],
    kd_search: list[Row],
    grid_search: list[Row],
    naive_search: list[Row],
    r_kd: list[Row],
    r_grid: list[Row],
    r_naive: list[Row],
):
    lines = [f"# Таблицы (среднее ± {SIGMA}σ, {BENCH_COUNT} прогонов, benchtime={BENCHTIME})\n\n"]

    def section(title: str, rows: list[Row], unit: str, scale: float):
        lines.append(f"## {title}\n\n")
        lines.append(f"| N | {unit} | B/op | ~ op/с |\n")
        lines.append("|---|-------|------|--------|\n")
        for n, mn, sn, mb, sb in rows:
            thr = 1e9 / mn if mn else 0
            lines.append(
                f"| {n:,} | {mn/scale:.2f} ± {sn/scale:.2f} | {mb:,.0f} | {thr:,.0f} |\n"
            )
        lines.append("\n")

    if kd_build:
        section("KD-tree Build", kd_build, "мс/op", 1e6)
    if kd_search:
        section("KD-tree Search (radius 500 km)", kd_search, "мкс/op", 1e3)
    if grid_search:
        section("Grid Search (radius 500 km)", grid_search, "мкс/op", 1e3)
    if naive_search:
        section("Naive Search (radius 500 km)", naive_search, "мкс/op", 1e3)

    if r_kd and r_grid and r_naive:
        lines.append("## Радиус vs время поиска (N = 100 000)\n\n")
        lines.append("| Радиус | KD, мкс | Grid, мкс | Naive, мс | KD/Naive |\n")
        lines.append("|--------|---------|-----------|-----------|----------|\n")
        for (r, kd_mn, kd_sn, _, _), (_, gr_mn, gr_sn, _, _), (_, na_mn, na_sn, _, _) in zip(r_kd, r_grid, r_naive):
            ratio = f"{na_mn/kd_mn:,.0f}×" if kd_mn > 0 else "—"
            lines.append(
                f"| {r/1000:.0f} km"
                f" | {kd_mn/1e3:.1f} ± {kd_sn/1e3:.1f}"
                f" | {gr_mn/1e3:.1f} ± {gr_sn/1e3:.1f}"
                f" | {na_mn/1e6:.1f} ± {na_sn/1e6:.1f}"
                f" | {ratio} |\n"
            )
        lines.append("\n")

    path.write_text("".join(lines), encoding="utf-8")


def main():
    try:
        import matplotlib  # noqa: F401
    except ImportError:
        print("pip install matplotlib", file=sys.stderr)
        sys.exit(1)

    FIG.mkdir(parents=True, exist_ok=True)
    RESULTS.mkdir(parents=True, exist_ok=True)

    saved = RESULTS / "bench.txt"
    if os.environ.get("REGEN_FROM_TXT") in ("1", "yes") and saved.is_file():
        raw = saved.read_text(encoding="utf-8")
    else:
        raw = run_bench()

    growth = parse_growth(raw)
    radius_raw = parse_radius(raw)

    def take(name: str) -> list[Row]:
        return agg(growth.get(name, {}))

    kd_build = take("BenchmarkGrowthKD_Build")
    kd_search = take("BenchmarkGrowthKD_Search")
    grid_search = take("BenchmarkGrowthGrid_Search")
    naive_search = take("BenchmarkGrowthNaive_Search")
    r_kd = agg(radius_raw.get("KD", {}))
    r_grid = agg(radius_raw.get("Grid", {}))
    r_naive = agg(radius_raw.get("Naive", {}))

    write_tables(
        RESULTS / "bench_tables.md",
        kd_build, kd_search, grid_search, naive_search,
        r_kd, r_grid, r_naive,
    )

    if kd_build:
        plot_single(kd_build, lambda r: (r[1] / 1e6, r[2] / 1e6),
                    "KD-tree Build — время vs N", "N, точек", "мс/op",
                    FIG / "growth_kd_build_time.png")
        plot_single(kd_build, lambda r: (r[3] / 1024, r[4] / 1024),
                    "KD-tree Build — память vs N", "N, точек", "KiB/op",
                    FIG / "growth_kd_build_mem.png")

    plot_multi(
        [
            ("KD-tree", kd_search, lambda r: (r[1] / 1e3, r[2] / 1e3)),
            ("Grid", grid_search, lambda r: (r[1] / 1e3, r[2] / 1e3)),
            ("Naive", naive_search, lambda r: (r[1] / 1e3, r[2] / 1e3)),
        ],
        "Search время vs N (radius = 500 km)",
        "N, точек", "мкс/op",
        FIG / "growth_search_compare_time.png", logy=True,
    )

    plot_multi(
        [
            ("KD-tree", kd_search, lambda r: (r[3] / 1024, r[4] / 1024)),
            ("Grid", grid_search, lambda r: (r[3] / 1024, r[4] / 1024)),
            ("Naive", naive_search, lambda r: (r[3] / 1024, r[4] / 1024)),
        ],
        "Search память vs N (radius = 500 km)",
        "N, точек", "KiB/op",
        FIG / "growth_search_compare_mem.png",
    )

    plot_multi(
        [
            ("KD-tree", r_kd, lambda r: (r[1] / 1e3, r[2] / 1e3)),
            ("Grid", r_grid, lambda r: (r[1] / 1e3, r[2] / 1e3)),
            ("Naive", r_naive, lambda r: (r[1] / 1e3, r[2] / 1e3)),
        ],
        "Search время vs radius (N = 100 000)",
        "radius, м", "мкс/op",
        FIG / "radius_compare_time.png", logy=True,
    )

    print(f"OK: figures/*.png + results/bench_tables.md ({BENCH_COUNT}×, {BENCHTIME})")


if __name__ == "__main__":
    main()
