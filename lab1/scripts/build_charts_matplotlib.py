#!/usr/bin/env python3
from __future__ import annotations

import json
import math
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
PROFILES = ROOT / "profiles"
GO_CACHE = ROOT / ".gocache"
MPL_CACHE = ROOT / ".mplconfig"
XDG_CACHE = ROOT / ".cache"

LINE_RE = re.compile(
    r"^(BenchmarkGrowth\S+)/N(\d+)-\d+\s+\d+\s+(\d+(?:\.\d+)?)\s+ns/op\s+(\d+)\s+B/op"
)
VALUE_RE = re.compile(r"^([0-9]+(?:\.[0-9]+)?)([A-Za-z]+)$")

BENCH_COUNT = int(os.environ.get("GROWTH_BENCH_COUNT", "5"))
BENCHTIME = os.environ.get("GROWTH_BENCHTIME", "200ms")
SIGMA = int(os.environ.get("GROWTH_SIGMA", "3"))
GROWTH_TIMEOUT = os.environ.get("GROWTH_TIMEOUT", "3600s")

MPL_CACHE.mkdir(parents=True, exist_ok=True)
XDG_CACHE.mkdir(parents=True, exist_ok=True)
os.environ.setdefault("MPLBACKEND", "Agg")
os.environ.setdefault("MPLCONFIGDIR", str(MPL_CACHE))
os.environ.setdefault("XDG_CACHE_HOME", str(XDG_CACHE))


PROFILE_NAMES = ("hashtable", "perfecthash", "lsh")


def go_env() -> dict[str, str]:
    env = os.environ.copy()
    GO_CACHE.mkdir(parents=True, exist_ok=True)
    env.setdefault("GOCACHE", str(GO_CACHE))
    return env


def run_cmd(cmd: list[str], *, env: dict[str, str] | None = None) -> str:
    proc = subprocess.run(
        cmd,
        cwd=ROOT,
        env=env or go_env(),
        capture_output=True,
        text=True,
    )
    out = proc.stdout + proc.stderr
    if proc.returncode != 0:
        print(out, file=sys.stderr)
        sys.exit(proc.returncode)
    return out


def run_go(cmd: list[str], *, env: dict[str, str] | None = None) -> str:
    return run_cmd(cmd, env=env)


def run_lab1_metrics() -> None:
    env = go_env()
    env["PHF_METRICS"] = "1"
    run_go(
        ["go", "test", "./perfecthash", "-run=^TestPHF_Metrics$", "-count=1", "-timeout=900s"],
        env=env,
    )
    env = go_env()
    env["LSH_METRICS"] = "1"
    run_go(
        ["go", "test", "./lsh", "-run=^TestLSH_Metrics$", "-count=1", "-timeout=900s"],
        env=env,
    )


def run_growth_bench() -> str:
    RESULTS.mkdir(parents=True, exist_ok=True)
    out = run_go(
        [
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
            f"-timeout={GROWTH_TIMEOUT}",
        ]
    )
    meta = f"# benchmeta count={BENCH_COUNT} benchtime={BENCHTIME}\n"
    (RESULTS / "growth_bench.txt").write_text(meta + out, encoding="utf-8")
    return out


def run_profiles() -> None:
    PROFILES.mkdir(parents=True, exist_ok=True)
    logs = []
    for name in PROFILE_NAMES:
        logs.append(run_go(["go", "run", "./cmd/profile", f"-only={name}"]))
    out = "\n".join(logs)
    (RESULTS / "profile_run.log").write_text(out, encoding="utf-8")


def run_flamegraphs() -> None:
    script = ROOT / "scripts" / "export_pprof_flame_html.sh"
    if not script.is_file():
        return
    base_port = int(os.environ.get("PPROF_PORT_BASE", "17890"))
    for offset, name in enumerate(PROFILE_NAMES):
        cpu_path = PROFILES / f"{name}_cpu.prof"
        if not cpu_path.is_file():
            continue
        html_path = FIG / f"pprof_{name}_flamegraph.html"
        env = go_env()
        env["PPROF_PORT"] = str(base_port + offset)
        run_cmd(["bash", str(script), str(cpu_path), str(html_path)], env=env)


def parse_samples(text: str) -> dict[str, dict[int, list[tuple[float, float]]]]:
    out: dict[str, dict[int, list[tuple[float, float]]]] = defaultdict(lambda: defaultdict(list))
    for raw_line in text.splitlines():
        line = raw_line.strip()
        if not line or line.startswith("#"):
            continue
        match = LINE_RE.match(line)
        if not match:
            continue
        name, n_s, ns_s, b_s = match.groups()
        out[name][int(n_s)].append((float(ns_s), float(b_s)))
    return {name: dict(rows) for name, rows in out.items()}


def agg(rows: dict[int, list[tuple[float, float]]]) -> list[tuple[int, float, float, float, float]]:
    out: list[tuple[int, float, float, float, float]] = []
    for n in sorted(rows):
        samples = rows[n]
        ns_vals = [sample[0] for sample in samples]
        b_vals = [sample[1] for sample in samples]
        ns_mean = statistics.mean(ns_vals)
        ns_std = statistics.stdev(ns_vals) if len(ns_vals) > 1 else 0.0
        b_mean = statistics.mean(b_vals)
        b_std = statistics.stdev(b_vals) if len(b_vals) > 1 else 0.0
        out.append((n, ns_mean, ns_std, b_mean, b_std))
    return out


def fgrp(value: float) -> str:
    return f"{value:,.0f}".replace(",", "\u202f")


def fgrp1(value: float) -> str:
    return f"{value:,.1f}".replace(",", "\u202f")


def fmt_ns(mean: float, std: float) -> str:
    spread = SIGMA * std
    if mean < 200:
        return f"{mean:.1f} ± {spread:.1f}"
    if spread < 1:
        return f"{fgrp(mean)} ± {spread:.2f}"
    if spread < 1000:
        return f"{fgrp(mean)} ± {spread:.1f}"
    return f"{fgrp(mean)} ± {fgrp(spread)}"


def fmt_ms(mean_ns: float, std_ns: float) -> str:
    return f"{mean_ns / 1e6:.2f} ± {(SIGMA * std_ns) / 1e6:.2f}"


def fmt_us(mean_ns: float, std_ns: float) -> str:
    return f"{mean_ns / 1e3:.2f} ± {(SIGMA * std_ns) / 1e3:.2f}"


def apply_plot_style() -> None:
    import matplotlib.pyplot as plt

    plt.rcParams.update(
        {
            "figure.facecolor": "white",
            "axes.facecolor": "#f8fafc",
            "axes.edgecolor": "#334155",
            "axes.linewidth": 0.8,
            "axes.grid": True,
            "grid.color": "#cbd5e1",
            "grid.linestyle": "--",
            "grid.linewidth": 0.6,
            "grid.alpha": 0.75,
            "axes.spines.top": False,
            "axes.spines.right": False,
            "font.size": 10,
            "axes.titlesize": 11,
            "axes.labelsize": 10,
            "legend.frameon": True,
            "legend.framealpha": 0.95,
            "legend.edgecolor": "#cbd5e1",
        }
    )


def human_n(value: int) -> str:
    if value >= 1_000_000:
        text = f"{value / 1_000_000:.1f}".rstrip("0").rstrip(".")
        return f"{text}M"
    if value >= 1000:
        if value % 1024 == 0:
            return f"{value // 1024}k"
        if value % 1000 == 0:
            return f"{value // 1000}k"
        text = f"{value / 1000:.1f}".rstrip("0").rstrip(".")
        return f"{text}k"
    return str(value)


def configure_series_axis(ax, xs: list[int], *, xlabel: str = "N") -> None:
    from matplotlib.ticker import MaxNLocator, NullLocator

    ax.set_xscale("log")
    ax.set_xlim(min(xs) * 0.93, max(xs) * 1.07)
    ax.set_xticks(xs, labels=[human_n(x) for x in xs])
    ax.xaxis.set_minor_locator(NullLocator())
    ax.yaxis.set_major_locator(MaxNLocator(nbins=6))
    ax.set_xlabel(xlabel)
    ax.tick_params(axis="x", labelrotation=0)
    ax.margins(y=0.08)


def plot_band(ax, rows, *, scale: float, label: str, color: str, marker: str = "o"):
    xs = [row[0] for row in rows]
    ym = [row[1] / scale for row in rows]
    ys = [SIGMA * row[2] / scale for row in rows]
    lo = [max(mean - spread, 0.0) for mean, spread in zip(ym, ys)]
    hi = [mean + spread for mean, spread in zip(ym, ys)]
    ax.fill_between(xs, lo, hi, alpha=0.18, color=color)
    ax.plot(xs, ym, marker + "-", color=color, linewidth=2.2, markersize=6, label=label, zorder=3)
    ax.errorbar(xs, ym, yerr=ys, fmt="none", ecolor=color, alpha=0.65, capsize=3, zorder=2)
    configure_series_axis(ax, xs)


def panel_finish(fig, path: Path, *, title: str | None = None) -> None:
    import matplotlib.pyplot as plt

    if title:
        fig.suptitle(title, fontsize=13, y=1.02)
    fig.tight_layout()
    fig.savefig(path)
    plt.close(fig)


def plot_disk_panel(series: dict[str, list[tuple[int, float, float, float, float]]]) -> None:
    import matplotlib.pyplot as plt

    fig, axes = plt.subplots(2, 2, figsize=(13, 8), dpi=150)
    items = [
        ("Insert", "us/op", 1e3, "#1d4ed8"),
        ("Update", "us/op", 1e3, "#0284c7"),
        ("Delete", "us/op", 1e3, "#c2410c"),
        ("Get", "ns/op", 1.0, "#047857"),
    ]
    for ax, (name, ylabel, scale, color) in zip(axes.flat, items):
        rows = series.get(name, [])
        if not rows:
            ax.set_visible(False)
            continue
        plot_band(ax, rows, scale=scale, label="mean", color=color)
        ax.set_title(name)
        ax.set_ylabel(ylabel)
        ax.legend(loc="upper left", fontsize=8)
    panel_finish(fig, FIG / "disk_growth_panel.png", title="Filesystem hash table")


def plot_ph_growth_panel(build_rows, get_rows) -> None:
    import matplotlib.pyplot as plt

    fig, axes = plt.subplots(1, 2, figsize=(12, 4.5), dpi=150)
    plot_band(axes[0], build_rows, scale=1e6, label="Build", color="#0f766e")
    axes[0].set_title("Build")
    axes[0].set_ylabel("ms/op")
    axes[0].legend(loc="upper left", fontsize=8)

    plot_band(axes[1], get_rows, scale=1.0, label="Get", color="#1d4ed8")
    axes[1].set_title("Get")
    axes[1].set_ylabel("ns/op")
    axes[1].legend(loc="upper left", fontsize=8)

    panel_finish(fig, FIG / "phf_growth_panel.png", title="Perfect hash growth benchmarks")


def plot_ph_metrics_panel(data: list[dict[str, float]]) -> None:
    import matplotlib.pyplot as plt

    xs = [row["n"] for row in data]
    bits = [row["displacement_bits_per_key"] for row in data]
    load = [row["load_factor"] for row in data]
    build = [row["build_ns_per_key_mean"] for row in data]
    build_sigma = [SIGMA * row["build_ns_per_key_std"] for row in data]
    query = [row["query_ns_mean"] for row in data]
    query_sigma = [SIGMA * row["query_ns_std"] for row in data]

    fig, axes = plt.subplots(2, 2, figsize=(12, 8), dpi=150)

    axes[0, 0].plot(xs, bits, "o-", color="#7c3aed", linewidth=2.2, markersize=6)
    axes[0, 0].set_title("Displacement bits/key")
    axes[0, 0].set_ylabel("bits/key")
    configure_series_axis(axes[0, 0], xs)

    axes[0, 1].plot(xs, load, "o-", color="#0891b2", linewidth=2.2, markersize=6)
    axes[0, 1].set_title("Load factor")
    axes[0, 1].set_ylabel("N / tableSize")
    configure_series_axis(axes[0, 1], xs)
    axes[0, 1].set_ylim(0, min(max(load) * 1.1, 1.05))

    lo = [max(mean - spread, 0.0) for mean, spread in zip(build, build_sigma)]
    hi = [mean + spread for mean, spread in zip(build, build_sigma)]
    axes[1, 0].fill_between(xs, lo, hi, alpha=0.18, color="#0f766e")
    axes[1, 0].plot(xs, build, "o-", color="#0f766e", linewidth=2.2, markersize=6)
    axes[1, 0].errorbar(xs, build, yerr=build_sigma, fmt="none", ecolor="#0f766e", alpha=0.65, capsize=3)
    axes[1, 0].set_title("Build, ns/key")
    axes[1, 0].set_ylabel("ns/key")
    configure_series_axis(axes[1, 0], xs)

    lo = [max(mean - spread, 0.0) for mean, spread in zip(query, query_sigma)]
    hi = [mean + spread for mean, spread in zip(query, query_sigma)]
    axes[1, 1].fill_between(xs, lo, hi, alpha=0.18, color="#1d4ed8")
    axes[1, 1].plot(xs, query, "o-", color="#1d4ed8", linewidth=2.2, markersize=6)
    axes[1, 1].errorbar(xs, query, yerr=query_sigma, fmt="none", ecolor="#1d4ed8", alpha=0.65, capsize=3)
    axes[1, 1].set_title("Get, ns/op")
    axes[1, 1].set_ylabel("ns/op")
    configure_series_axis(axes[1, 1], xs)

    panel_finish(fig, FIG / "phf_metrics_panel.png", title="Perfect hash metadata")


def plot_lsh_growth_panel(build_rows, add_rows, find_rows, naive_rows) -> None:
    import matplotlib.pyplot as plt

    fig, axes = plt.subplots(1, 3, figsize=(15, 4.8), dpi=150)

    plot_band(axes[0], build_rows, scale=1e6, label="Build", color="#0f766e")
    axes[0].set_title("Build")
    axes[0].set_ylabel("ms/op")
    axes[0].legend(loc="upper left", fontsize=8)

    plot_band(axes[1], add_rows, scale=1.0, label="Add", color="#0284c7")
    axes[1].set_title("Add")
    axes[1].set_ylabel("ns/op")
    axes[1].legend(loc="upper left", fontsize=8)

    plot_band(axes[2], find_rows, scale=1e6, label="LSH find", color="#1d4ed8")
    plot_band(axes[2], naive_rows, scale=1e6, label="Naive scan", color="#c2410c", marker="s")
    axes[2].set_title("Find vs naive")
    axes[2].set_ylabel("ms/op")
    axes[2].legend(loc="upper left", fontsize=8)

    panel_finish(fig, FIG / "lsh_growth_panel.png", title="LSH growth benchmarks")


def plot_lsh_metrics_panel(data: list[dict[str, float]]) -> None:
    import matplotlib.pyplot as plt

    xs = [row["n"] for row in data]
    recall = [row["recall_mean"] for row in data]
    recall_sigma = [SIGMA * row["recall_std"] for row in data]
    precision = [row["precision_mean"] for row in data]
    precision_sigma = [SIGMA * row["precision_std"] for row in data]
    cand = [row["candidate_ratio_mean"] * 100 for row in data]
    cand_sigma = [SIGMA * row["candidate_ratio_std"] * 100 for row in data]
    speedup = [row["speedup_mean"] for row in data]
    speedup_sigma = [SIGMA * row["speedup_std"] for row in data]

    fig, axes = plt.subplots(1, 2, figsize=(12, 4.6), dpi=150)

    lo = [max(mean - spread, 0.0) for mean, spread in zip(speedup, speedup_sigma)]
    hi = [mean + spread for mean, spread in zip(speedup, speedup_sigma)]
    axes[0].fill_between(xs, lo, hi, alpha=0.18, color="#047857")
    axes[0].plot(xs, speedup, "o-", color="#047857", linewidth=2.2, markersize=6)
    axes[0].errorbar(xs, speedup, yerr=speedup_sigma, fmt="none", ecolor="#047857", alpha=0.65, capsize=3)
    axes[0].axhline(1.0, color="#64748b", linestyle="--", linewidth=1)
    axes[0].set_title("Speedup vs naive")
    axes[0].set_ylabel("x")
    configure_series_axis(axes[0], xs)

    axes[1].fill_between(
        xs,
        [max(mean - spread, 0.0) for mean, spread in zip(recall, recall_sigma)],
        [mean + spread for mean, spread in zip(recall, recall_sigma)],
        alpha=0.12,
        color="#1d4ed8",
    )
    axes[1].plot(xs, recall, "o-", color="#1d4ed8", linewidth=2.0, markersize=5, label="recall")
    axes[1].fill_between(
        xs,
        [max(mean - spread, 0.0) for mean, spread in zip(precision, precision_sigma)],
        [mean + spread for mean, spread in zip(precision, precision_sigma)],
        alpha=0.12,
        color="#0ea5e9",
    )
    axes[1].plot(xs, precision, "s-", color="#0ea5e9", linewidth=1.9, markersize=5, label="precision")
    axes[1].set_title("Accuracy and candidates")
    axes[1].set_ylabel("recall / precision")
    axes[1].set_ylim(0, 1.05)
    configure_series_axis(axes[1], xs)
    twin = axes[1].twinx()
    twin.fill_between(
        xs,
        [max(mean - spread, 0.0) for mean, spread in zip(cand, cand_sigma)],
        [mean + spread for mean, spread in zip(cand, cand_sigma)],
        alpha=0.12,
        color="#b91c1c",
    )
    twin.plot(xs, cand, "^-", color="#b91c1c", linewidth=1.8, markersize=5, label="cand / all, %")
    twin.set_ylabel("cand / all, %")
    lines_a, labels_a = axes[1].get_legend_handles_labels()
    lines_b, labels_b = twin.get_legend_handles_labels()
    axes[1].legend(lines_a + lines_b, labels_a + labels_b, loc="center right", fontsize=7)

    panel_finish(fig, FIG / "lsh_metrics_panel.png", title="LSH quality metrics")


def parse_value(text: str) -> float | None:
    text = text.strip()
    if text == "0":
        return 0.0
    match = VALUE_RE.match(text)
    if not match:
        return None
    value = float(match.group(1))
    unit = match.group(2)
    if unit == "ns":
        return value / 1e9
    if unit == "us":
        return value / 1e6
    if unit == "ms":
        return value / 1e3
    if unit == "s":
        return value
    if unit == "B":
        return value
    if unit == "kB":
        return value * 1024
    if unit == "MB":
        return value * 1024 * 1024
    if unit == "GB":
        return value * 1024 * 1024 * 1024
    if unit == "TB":
        return value * 1024 * 1024 * 1024 * 1024
    return None


def should_skip_profile_name(name: str, *, is_memory: bool = False) -> bool:
    blocked = (
        "compress/flate.",
        "main.main",
        "main.prepare",
        "main.run",
        "main.writeCPU",
        "main.writeMem",
        "runtime/pprof.",
        "runtime.main",
        "runtime.goexit",
        "runtime.allocm",
        "runtime.findRunnable",
        "runtime.mPark",
        "runtime.newm",
        "runtime.notesleep",
        "runtime.park_m",
        "runtime.resetspinning",
        "runtime.schedule",
        "runtime.startm",
        "runtime.systemstack",
        "runtime.gcBgMarkWorker",
        "runtime.gcDrain",
        "runtime.gcDrainMarkWorker",
        "runtime.greyobject",
        "runtime.scanobject",
        "runtime.madvise",
        "runtime.(*mheap).",
        "runtime.allocm",
        "runtime.sysUsed",
        "runtime.sysUsedOS",
        "runtime.sysUnused",
        "runtime.sysUnusedOS",
    )
    if name.startswith(blocked):
        return True
    if is_memory and name.startswith(
        (
            "algo/lsh.GenerateDataset",
            "fmt.",
            "runtime.mcall",
            "runtime.mstart",
            "runtime.mstart0",
            "runtime.mstart1",
            "runtime.wakep",
        )
    ):
        return True
    return False


def is_preferred_profile_name(name: str) -> bool:
    return name.startswith(
        (
            "algo/",
            "os.",
            "syscall.",
            "internal/poll.",
            "strconv.",
            "strings.",
            "math/rand.",
        )
    )


def parse_pprof_top(
    profile_path: Path,
    *,
    sample_index: str | None = None,
    cumulative: bool = True,
) -> list[tuple[str, float]]:
    cmd = ["go", "tool", "pprof", "-top"]
    if cumulative:
        cmd.append("-cum")
    cmd.append(str(profile_path))
    if sample_index:
        cmd.insert(3, f"-sample_index={sample_index}")
    text = run_go(cmd)
    best: dict[str, float] = {}
    for raw_line in text.splitlines():
        line = raw_line.strip()
        if not line:
            continue
        if line.startswith("File:") or line.startswith("Type:"):
            continue
        if line.startswith("Time:") or line.startswith("Duration:"):
            continue
        if line.startswith("Showing nodes") or line.startswith("Dropped"):
            continue
        if "flat" in line and "cum" in line and "flat%" in line:
            continue
        parts = line.split()
        if len(parts) < 6:
            continue
        value = parse_value(parts[3] if cumulative else parts[0])
        if value is None:
            continue
        name = " ".join(parts[5:])
        if should_skip_profile_name(name, is_memory=sample_index is not None):
            continue
        if name not in best or value > best[name]:
            best[name] = value
    ranked = sorted(best.items(), key=lambda item: -item[1])
    preferred = [item for item in ranked if is_preferred_profile_name(item[0])]
    rest = [item for item in ranked if not is_preferred_profile_name(item[0])]
    return (preferred + rest)[:8]


def short_label(name: str) -> str:
    if len(name) <= 48:
        return name
    return name[:45] + "..."


def plot_profile_panel(name: str, cpu_items, mem_items) -> None:
    import matplotlib.pyplot as plt

    fig, axes = plt.subplots(1, 2, figsize=(13, 5), dpi=150)
    cpu_labels = [short_label(item[0]) for item in reversed(cpu_items)]
    cpu_vals = [item[1] for item in reversed(cpu_items)]
    axes[0].barh(cpu_labels, cpu_vals, color="#1d4ed8")
    axes[0].set_title("CPU hotspots")
    axes[0].set_xlabel("cum seconds")

    mem_labels = [short_label(item[0]) for item in reversed(mem_items)]
    mem_vals = [item[1] / (1024 * 1024) for item in reversed(mem_items)]
    axes[1].barh(mem_labels, mem_vals, color="#c2410c")
    axes[1].set_title("Allocation hotspots")
    axes[1].set_xlabel("cum MiB (alloc_space)")

    panel_finish(fig, FIG / f"profile_{name}.png", title=f"{name} profile summary")


def collect_profiles() -> dict[str, dict[str, list[tuple[str, float]]]]:
    profiles = {}
    for name in PROFILE_NAMES:
        cpu_path = PROFILES / f"{name}_cpu.prof"
        mem_path = PROFILES / f"{name}_mem.prof"
        if not cpu_path.is_file() or not mem_path.is_file():
            continue
        profiles[name] = {
            "cpu": parse_pprof_top(cpu_path, cumulative=True),
            "mem": parse_pprof_top(mem_path, sample_index="alloc_space", cumulative=True),
        }
    return profiles


def profile_md_block(title: str, items: list[tuple[str, float]], *, is_memory: bool) -> list[str]:
    lines = [f"### {title}\n\n"]
    lines.append("| function | cumulative |\n")
    lines.append("|---|---|\n")
    shown = 0
    for name, value in items:
        if is_memory:
            mib = value / (1024 * 1024)
            if mib < 0.01:
                continue
            lines.append(f"| `{name}` | {mib:.2f} MiB |\n")
        else:
            if value < 0.01:
                continue
            lines.append(f"| `{name}` | {value:.2f} s |\n")
        shown += 1
        if shown == 5:
            break
    lines.append("\n")
    return lines


def write_growth_tables(path: Path, sections: dict[str, list[tuple[int, float, float, float, float]]], ph_metrics, lsh_metrics):
    lines: list[str] = []

    def write_ns_section(title: str, rows):
        lines.append(f"## {title}\n\n")
        lines.append(f"| N | ns/op (±{SIGMA}σ) | B/op (±{SIGMA}σ) | ~ op/s |\n")
        lines.append("|---|---|---|---|\n")
        for n, mean, std, b_mean, b_std in rows:
            lines.append(
                f"| {n} | {fmt_ns(mean, std)} | {fgrp(b_mean)} ± {fgrp1(SIGMA * b_std)} | {fgrp(1e9 / mean if mean else 0)} |\n"
            )
        lines.append("\n")

    def write_ms_section(title: str, rows):
        lines.append(f"## {title}\n\n")
        lines.append(f"| N | ms/op (±{SIGMA}σ) | KiB/op (±{SIGMA}σ) | ~ op/s |\n")
        lines.append("|---|---|---|---|\n")
        for n, mean, std, b_mean, b_std in rows:
            lines.append(
                f"| {n} | {fmt_ms(mean, std)} | {fgrp1(b_mean / 1024)} ± {fgrp1((SIGMA * b_std) / 1024)} | {fgrp(1e9 / mean if mean else 0)} |\n"
            )
        lines.append("\n")

    for name in ("Hashtable: Insert", "Hashtable: Update", "Hashtable: Delete", "Hashtable: Get"):
        rows = sections.get(name, [])
        if rows:
            write_ns_section(name, rows)

    for name in ("Perfect hash: Build", "Perfect hash: Get"):
        rows = sections.get(name, [])
        if rows:
            if name.endswith("Build"):
                write_ms_section(name, rows)
            else:
                write_ns_section(name, rows)

    for name in ("LSH: Build", "LSH: Add", "LSH: Find", "LSH: Naive"):
        rows = sections.get(name, [])
        if rows:
            if name.endswith(("Build", "Find", "Naive")):
                write_ms_section(name, rows)
            else:
                write_ns_section(name, rows)

    if ph_metrics:
        lines.append("## Perfect hash: Summary\n\n")
        lines.append(f"| N | load factor | disp. bits/key | build ns/key (±{SIGMA}σ) | get ns/op (±{SIGMA}σ) |\n")
        lines.append("|---|---|---|---|---|\n")
        for row in ph_metrics:
            lines.append(
                f"| {row['n']} | {row['load_factor']:.3f} | {row['displacement_bits_per_key']:.2f} | "
                f"{row['build_ns_per_key_mean']:.1f} ± {SIGMA * row['build_ns_per_key_std']:.1f} | "
                f"{row['query_ns_mean']:.1f} ± {SIGMA * row['query_ns_std']:.1f} |\n"
            )
        lines.append("\n")

    if lsh_metrics:
        lines.append("## LSH: Summary\n\n")
        lines.append(
            f"| N | recall (±{SIGMA}σ) | precision (±{SIGMA}σ) | cand/all (±{SIGMA}σ) | build ms (±{SIGMA}σ) | add ns/op (±{SIGMA}σ) | find ms (±{SIGMA}σ) | naive ms (±{SIGMA}σ) | speedup (±{SIGMA}σ) |\n"
        )
        lines.append("|---|---|---|---|---|---|---|---|---|\n")
        for row in lsh_metrics:
            lines.append(
                f"| {row['n']} | {row['recall_mean']:.3f} ± {SIGMA * row['recall_std']:.3f} | "
                f"{row['precision_mean']:.3f} ± {SIGMA * row['precision_std']:.3f} | "
                f"{row['candidate_ratio_mean'] * 100:.3f}% ± {SIGMA * row['candidate_ratio_std'] * 100:.3f}% | "
                f"{row['build_ms_mean']:.2f} ± {SIGMA * row['build_ms_std']:.2f} | "
                f"{row['add_ns_mean']:.1f} ± {SIGMA * row['add_ns_std']:.1f} | "
                f"{row['find_ms_mean']:.2f} ± {SIGMA * row['find_ms_std']:.2f} | "
                f"{row['naive_ms_mean']:.2f} ± {SIGMA * row['naive_ms_std']:.2f} | "
                f"{row['speedup_mean']:.2f} ± {SIGMA * row['speedup_std']:.2f} |\n"
            )
        lines.append("\n")

    path.write_text("".join(lines), encoding="utf-8")


def write_profile_tables(path: Path, profiles) -> None:
    lines: list[str] = []
    names = {
        "hashtable": "Hashtable profile",
        "perfecthash": "Perfect hash profile",
        "lsh": "LSH profile",
    }
    for key in ("hashtable", "perfecthash", "lsh"):
        info = profiles.get(key)
        if not info:
            continue
        lines.append(f"## {names[key]}\n\n")
        lines.extend(profile_md_block("CPU", info["cpu"], is_memory=False))
        lines.extend(profile_md_block("Allocated bytes (alloc_space)", info["mem"], is_memory=True))
    path.write_text("".join(lines), encoding="utf-8")


def split_md_h2_sections(text: str) -> dict[str, str]:
    sections: dict[str, str] = {}
    for block in ("\n" + text).split("\n## ")[1:]:
        pos = block.find("\n")
        if pos == -1:
            continue
        sections[block[:pos].strip()] = block[pos + 1 :].strip()
    return sections


def sync_readme() -> None:
    readme_path = ROOT / "README.md"
    if not readme_path.is_file():
        return
    text = readme_path.read_text(encoding="utf-8")
    growth_sections = split_md_h2_sections((RESULTS / "growth_bench_tables.md").read_text(encoding="utf-8"))
    profile_sections = split_md_h2_sections((RESULTS / "profile_tables.md").read_text(encoding="utf-8"))

    blocks = {
        "TBL_DISK": "\n\n".join(
            filter(
                None,
                [
                    "**Insert**\n\n" + growth_sections.get("Hashtable: Insert", ""),
                    "**Update**\n\n" + growth_sections.get("Hashtable: Update", ""),
                    "**Delete**\n\n" + growth_sections.get("Hashtable: Delete", ""),
                    "**Get**\n\n" + growth_sections.get("Hashtable: Get", ""),
                ],
            )
        ),
        "TBL_CHD_BENCH": "\n\n".join(
            filter(
                None,
                [
                    "**Build**\n\n" + growth_sections.get("Perfect hash: Build", ""),
                    "**Get**\n\n" + growth_sections.get("Perfect hash: Get", ""),
                ],
            )
        ),
        "TBL_CHD_METRICS": growth_sections.get("Perfect hash: Summary", ""),
        "TBL_LSH_BENCH": "\n\n".join(
            filter(
                None,
                [
                    "**Build**\n\n" + growth_sections.get("LSH: Build", ""),
                    "**Add**\n\n" + growth_sections.get("LSH: Add", ""),
                    "**Find**\n\n" + growth_sections.get("LSH: Find", ""),
                    "**Naive**\n\n" + growth_sections.get("LSH: Naive", ""),
                ],
            )
        ),
        "TBL_LSH_METRICS": growth_sections.get("LSH: Summary", ""),
        "TBL_PROF_HASH": profile_sections.get("Hashtable profile", ""),
        "TBL_PROF_CHD": profile_sections.get("Perfect hash profile", ""),
        "TBL_PROF_LSH": profile_sections.get("LSH profile", ""),
    }

    for key, block in blocks.items():
        start = f"<!-- {key} -->"
        end = f"<!-- /{key} -->"
        i = text.find(start)
        j = text.find(end)
        if i < 0 or j < 0 or j <= i:
            continue
        text = text[: i + len(start)] + "\n\n" + block.strip() + "\n\n" + text[j:]
    readme_path.write_text(text, encoding="utf-8")


def main() -> None:
    try:
        import matplotlib  # noqa: F401
    except ImportError:
        print("Install matplotlib: python3 -m pip install -r lab1/scripts/requirements.txt", file=sys.stderr)
        sys.exit(1)

    FIG.mkdir(parents=True, exist_ok=True)
    RESULTS.mkdir(parents=True, exist_ok=True)
    apply_plot_style()

    regen = os.environ.get("REGEN_FROM_TXT", "").lower() in {"1", "true", "yes"}
    skip_metrics = os.environ.get("SKIP_METRICS", "").lower() in {"1", "true", "yes"}
    skip_profiles = os.environ.get("SKIP_PROFILES", "").lower() in {"1", "true", "yes"}
    skip_flamegraphs = os.environ.get("SKIP_FLAMEGRAPHS", "").lower() in {"1", "true", "yes"}

    if regen:
        raw = (RESULTS / "growth_bench.txt").read_text(encoding="utf-8")
    else:
        if not skip_metrics:
            run_lab1_metrics()
        raw = run_growth_bench()
        if not skip_profiles:
            run_profiles()

    samples = parse_samples(raw)
    series = {
        "Hashtable: Insert": agg(samples.get("BenchmarkGrowthDisk_Insert", {})),
        "Hashtable: Update": agg(samples.get("BenchmarkGrowthDisk_Update", {})),
        "Hashtable: Delete": agg(samples.get("BenchmarkGrowthDisk_Delete", {})),
        "Hashtable: Get": agg(samples.get("BenchmarkGrowthDisk_Get", {})),
        "Perfect hash: Build": agg(samples.get("BenchmarkGrowthPH_Build", {})),
        "Perfect hash: Get": agg(samples.get("BenchmarkGrowthPH_Get", {})),
        "LSH: Build": agg(samples.get("BenchmarkGrowthLSH_Build", {})),
        "LSH: Add": agg(samples.get("BenchmarkGrowthLSH_Add", {})),
        "LSH: Find": agg(samples.get("BenchmarkGrowthLSH_Find", {})),
        "LSH: Naive": agg(samples.get("BenchmarkGrowthLSH_Naive", {})),
    }

    ph_metrics = []
    ph_path = RESULTS / "phf_metrics.json"
    if ph_path.is_file():
        ph_metrics = json.loads(ph_path.read_text(encoding="utf-8"))
    lsh_metrics = []
    lsh_path = RESULTS / "lsh_metrics.json"
    if lsh_path.is_file():
        lsh_metrics = json.loads(lsh_path.read_text(encoding="utf-8"))

    profiles = collect_profiles()
    if profiles and not skip_flamegraphs:
        run_flamegraphs()

    write_growth_tables(RESULTS / "growth_bench_tables.md", series, ph_metrics, lsh_metrics)
    write_profile_tables(RESULTS / "profile_tables.md", profiles)

    plot_disk_panel(
        {
            "Insert": series["Hashtable: Insert"],
            "Update": series["Hashtable: Update"],
            "Delete": series["Hashtable: Delete"],
            "Get": series["Hashtable: Get"],
        }
    )
    plot_ph_growth_panel(series["Perfect hash: Build"], series["Perfect hash: Get"])
    if ph_metrics:
        plot_ph_metrics_panel(ph_metrics)
    plot_lsh_growth_panel(
        series["LSH: Build"],
        series["LSH: Add"],
        series["LSH: Find"],
        series["LSH: Naive"],
    )
    if lsh_metrics:
        plot_lsh_metrics_panel(lsh_metrics)
    for name, info in profiles.items():
        plot_profile_panel(name, info["cpu"], info["mem"])

    if os.environ.get("SYNC_README", "").lower() in {"1", "true", "yes"}:
        sync_readme()

    print(
        f"OK: figures, results/growth_bench_tables.md, results/profile_tables.md "
        f"({BENCH_COUNT}x, {BENCHTIME}, ±{SIGMA}σ)"
    )


if __name__ == "__main__":
    main()
