#!/usr/bin/env python3
"""go test BenchmarkGrowth → parse → matplotlib → figures/growth_*.png. README уже ссылается на эти файлы."""
from __future__ import annotations

import re
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


def run_bench() -> str:
    RESULTS.mkdir(parents=True, exist_ok=True)
    cmd = [
        "go",
        "test",
        "./hashtable",
        "./perfecthash",
        "./lsh",
        "-run=^$",
        "-bench=BenchmarkGrowth",
        "-benchmem",
        "-benchtime=150ms",
        "-count=1",
        "-timeout=180s",
    ]
    p = subprocess.run(cmd, cwd=ROOT, capture_output=True, text=True)
    out = p.stdout + p.stderr
    (RESULTS / "growth_bench.txt").write_text(out, encoding="utf-8")
    if p.returncode != 0:
        print(out, file=sys.stderr)
        sys.exit(p.returncode)
    return out


def parse(text: str) -> dict[str, list[tuple[int, float, float]]]:
    m: dict[str, list[tuple[int, float, float]]] = defaultdict(list)
    for line in text.splitlines():
        line = line.strip()
        mo = LINE_RE.match(line)
        if not mo:
            continue
        name, n_s, ns_s, b_s = mo.groups()
        m[name].append((int(n_s), float(ns_s), float(b_s)))
    for k in m:
        m[k].sort(key=lambda t: t[0])
    return dict(m)


def plot_xy(
    xs, ys, title: str, xlab: str, ylab: str, path: Path, logx: bool = True, logy: bool = False
):
    import matplotlib.pyplot as plt

    fig, ax = plt.subplots(figsize=(10, 5), dpi=120)
    ax.plot(xs, ys, "o-", linewidth=2, markersize=6, color="#2563eb")
    ax.set_title(title)
    ax.set_xlabel(xlab)
    ax.set_ylabel(ylab)
    if logx:
        ax.set_xscale("log")
    if logy:
        ax.set_yscale("log")
    ax.grid(True, alpha=0.3)
    fig.tight_layout()
    fig.savefig(path)
    plt.close(fig)


def plot_dual(series_a, series_b, title, xlab, ylab, path: Path, label_a: str, label_b: str):
    import matplotlib.pyplot as plt

    fig, ax = plt.subplots(figsize=(10, 5), dpi=120)
    if series_a:
        xs = [t[0] for t in series_a]
        ys = [t[1] / 1e6 for t in series_a]
        ax.plot(xs, ys, "o-", label=label_a, linewidth=2, color="#2563eb")
    if series_b:
        xs = [t[0] for t in series_b]
        ys = [t[1] / 1e6 for t in series_b]
        ax.plot(xs, ys, "s-", label=label_b, linewidth=2, color="#d97706")
    ax.set_title(title)
    ax.set_xlabel(xlab)
    ax.set_ylabel(ylab)
    ax.set_xscale("log")
    ax.set_yscale("log")
    ax.legend()
    ax.grid(True, alpha=0.3)
    fig.tight_layout()
    fig.savefig(path)
    plt.close(fig)


def plot_multi(groups: list[tuple[str, list]], title, xlab, ylab, path: Path):
    import matplotlib.pyplot as plt

    fig, ax = plt.subplots(figsize=(10, 5), dpi=120)
    colors = ["#2563eb", "#059669", "#d97706"]
    for i, (label, series) in enumerate(groups):
        if not series:
            continue
        xs = [t[0] for t in series]
        ys = [t[1] / 1e3 for t in series]
        ax.plot(xs, ys, "o-", label=label, linewidth=2, color=colors[i % len(colors)])
    ax.set_title(title)
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
    s = parse(raw)

    def take(name):
        return s.get(name, [])

    ph_b = take("BenchmarkGrowthPH_Build")
    ph_g = take("BenchmarkGrowthPH_Get")
    lsh = take("BenchmarkGrowthLSH_Find")
    naive = take("BenchmarkGrowthLSH_Naive")
    d_set = take("BenchmarkGrowthDisk_Set")
    d_get = take("BenchmarkGrowthDisk_Get")

    if ph_b:
        xs, ys = zip(*[(t[0], t[1] / 1e6) for t in ph_b])
        plot_xy(xs, ys, "Perfect hash: Build — время", "N, ключей", "мс/op", FIG / "growth_ph_build_time.png")
        xs, ys = zip(*[(t[0], t[2] / 1024) for t in ph_b])
        plot_xy(
            xs, ys, "Perfect hash: Build — память", "N, ключей", "KiB/op", FIG / "growth_ph_build_mem.png"
        )

    if ph_g:
        xs, ys = zip(*[(t[0], t[1]) for t in ph_g])
        plot_xy(xs, ys, "Perfect hash: Get — время", "N, ключей", "нс/op", FIG / "growth_ph_get_time.png")
        xs, ys = zip(*[(t[0], t[2] / 1024) for t in ph_g])
        plot_xy(xs, ys, "Perfect hash: Get — память", "N, ключей", "KiB/op", FIG / "growth_ph_get_mem.png")

    if lsh:
        xs, ys = zip(*[(t[0], t[1] / 1e6) for t in lsh])
        plot_xy(
            xs, ys, "LSH FindDuplicates — время", "N, точек", "мс/op", FIG / "growth_lsh_find_time.png"
        )
        xs, ys = zip(*[(t[0], t[2] / 1024) for t in lsh])
        plot_xy(
            xs, ys, "LSH FindDuplicates — память", "N, точек", "KiB/op", FIG / "growth_lsh_find_mem.png"
        )

    if naive:
        xs, ys = zip(*[(t[0], t[1] / 1e6) for t in naive])
        plot_xy(xs, ys, "Наивный скан — время", "N, точек", "мс/op", FIG / "growth_naive_time.png")
        xs, ys = zip(*[(t[0], t[2] / 1024) for t in naive])
        plot_xy(xs, ys, "Наивный скан — память", "N, точек", "KiB/op", FIG / "growth_naive_mem.png")

    plot_dual(
        lsh,
        naive,
        "LSH vs наивный (время)",
        "N, точек",
        "мс/op",
        FIG / "growth_dup_compare_time.png",
        "LSH",
        "Naive",
    )

    plot_multi(
        [
            ("Set", d_set),
            ("Get", d_get),
        ],
        "Дисковая таблица: Set и Get",
        "N",
        "мкс/op",
        FIG / "growth_disk_time.png",
    )

    print("OK: figures/growth_*.png")


if __name__ == "__main__":
    main()
