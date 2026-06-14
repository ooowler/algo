#!/usr/bin/env python3
from __future__ import annotations

import os
import re
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
RESULTS = ROOT / "results"
FIG = ROOT / "figures"
MPL_CACHE = ROOT / ".mplconfig"
XDG_CACHE = ROOT / ".cache"

MPL_CACHE.mkdir(parents=True, exist_ok=True)
XDG_CACHE.mkdir(parents=True, exist_ok=True)
os.environ.setdefault("MPLBACKEND", "Agg")
os.environ.setdefault("MPLCONFIGDIR", str(MPL_CACHE))
os.environ.setdefault("XDG_CACHE_HOME", str(XDG_CACHE))


def main() -> None:
    FIG.mkdir(parents=True, exist_ok=True)
    metrics, rows = parse_stress(RESULTS / "stress.md")
    profile = parse_profile(RESULTS / "profile_cpu.txt")
    apply_style()
    plot_latency_percentiles(rows)
    plot_tail_range(rows)
    plot_throughput(rows)
    plot_selectivity_latency(rows)
    plot_index_footprint(metrics)
    plot_profile_top(profile)


def parse_stress(path: Path) -> tuple[dict[str, str], list[dict[str, object]]]:
    metrics: dict[str, str] = {}
    rows: list[dict[str, object]] = []
    for line in path.read_text(encoding="utf-8").splitlines():
        cols = split_md_row(line)
        if len(cols) == 2 and cols[0] not in ("Metric", "---"):
            metrics[cols[0]] = cols[1].strip("`")
        if len(cols) >= 10 and cols[0].startswith("`"):
            name = cols[0].strip("`")
            rows.append(
                {
                    "name": name,
                    "query": cols[1].strip("`"),
                    "l0": int(cols[2]),
                    "returned": int(cols[3]),
                    "min": float(cols[4]),
                    "avg": float(cols[5]),
                    "p95": float(cols[6]),
                    "max": float(cols[7]),
                    "qps": float(cols[8]),
                    "error": cols[9],
                }
            )
    return metrics, rows


def parse_profile(path: Path) -> list[dict[str, object]]:
    rows: list[dict[str, object]] = []
    for line in path.read_text(encoding="utf-8").splitlines():
        fields = line.split()
        if len(fields) < 6 or not fields[0].endswith("ms"):
            continue
        rows.append(
            {
                "flat": float(fields[0][:-2]),
                "flat_pct": float(fields[1].rstrip("%")),
                "cum": float(fields[3][:-2]),
                "cum_pct": float(fields[4].rstrip("%")),
                "function": " ".join(fields[5:]),
            }
        )
        if len(rows) == 10:
            break
    return rows


def split_md_row(line: str) -> list[str]:
    line = line.strip()
    if not line.startswith("|") or not line.endswith("|"):
        return []
    return [part.strip() for part in line.strip("|").split("|")]


def apply_style() -> None:
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
            "axes.titlesize": 13,
            "axes.labelsize": 10,
            "legend.frameon": True,
            "legend.framealpha": 0.96,
            "legend.edgecolor": "#cbd5e1",
            "savefig.dpi": 170,
            "savefig.bbox": "tight",
        }
    )


def valid_rows(rows: list[dict[str, object]]) -> list[dict[str, object]]:
    return [row for row in rows if not row["error"]]


def nontrivial_rows(rows: list[dict[str, object]]) -> list[dict[str, object]]:
    return [row for row in valid_rows(rows) if row["name"] != "missing"]


def plot_latency_percentiles(rows: list[dict[str, object]]) -> None:
    import matplotlib.pyplot as plt
    import numpy as np

    data = nontrivial_rows(rows)
    labels = [short(str(row["name"])) for row in data]
    y = np.arange(len(data))
    height = 0.32

    fig, ax = plt.subplots(figsize=(11.8, 6.0))
    ax.barh(y - height / 2, [row["avg"] for row in data], height, label="avg", color="#2563eb")
    ax.barh(y + height / 2, [row["p95"] for row in data], height, label="p95", color="#f97316")
    ax.set_yticks(y, labels)
    ax.invert_yaxis()
    ax.set_xlabel("milliseconds")
    ax.set_title("Avg and p95 latency by query scenario")
    ax.legend(loc="upper center", bbox_to_anchor=(0.86, -0.08), ncol=2)
    for i, row in enumerate(data):
        ax.text(float(row["p95"]) + 0.025, i + 0.04, f'{row["p95"]:.3f}', va="center", fontsize=8, color="#9a3412")
    finish(fig, "stress_latency_percentiles.png")


def plot_tail_range(rows: list[dict[str, object]]) -> None:
    import matplotlib.pyplot as plt
    import numpy as np

    data = nontrivial_rows(rows)
    labels = [short(str(row["name"])) for row in data]
    y = np.arange(len(data))

    fig, ax = plt.subplots(figsize=(11.8, 5.9))
    for i, row in enumerate(data):
        ax.plot([row["min"], row["max"]], [i, i], color="#94a3b8", linewidth=2, zorder=1, label="min..max" if i == 0 else None)
    ax.scatter([row["avg"] for row in data], y, label="avg", color="#2563eb", s=46, zorder=3)
    ax.scatter([row["p95"] for row in data], y, label="p95", color="#f97316", s=46, zorder=3)
    ax.set_yticks(y, labels)
    ax.invert_yaxis()
    ax.set_xlabel("milliseconds, min..max range")
    ax.set_title("Tail latency spread")
    ax.legend(loc="lower right")
    finish(fig, "stress_tail_range.png")


def plot_throughput(rows: list[dict[str, object]]) -> None:
    import matplotlib.pyplot as plt
    import numpy as np

    data = valid_rows(rows)
    data = sorted(data, key=lambda row: float(row["qps"]))
    labels = [short(str(row["name"])) for row in data]
    y = np.arange(len(data))

    fig, ax = plt.subplots(figsize=(11.8, 6.1))
    colors = ["#059669" if row["name"] != "missing" else "#64748b" for row in data]
    ax.barh(y, [row["qps"] for row in data], color=colors)
    ax.set_xscale("log")
    ax.set_yticks(y, labels)
    ax.set_xlabel("queries per second, log scale")
    ax.set_title("Throughput by query scenario")
    for i, row in enumerate(data):
        ax.text(float(row["qps"]) * 1.05, i, f'{row["qps"]:.0f}', va="center", fontsize=8)
    finish(fig, "stress_throughput_qps.png")


def plot_selectivity_latency(rows: list[dict[str, object]]) -> None:
    import matplotlib.pyplot as plt

    data = nontrivial_rows(rows)
    fig, ax = plt.subplots(figsize=(10.8, 6.1))
    xs = [int(row["l0"]) for row in data]
    ys = [float(row["p95"]) for row in data]
    sizes = [max(70, min(520, int(row["returned"]) * 34)) for row in data]
    ax.scatter(xs, ys, s=sizes, color="#7c3aed", alpha=0.78, edgecolor="white", linewidth=1.2)
    offsets = {
        "term_common": (8, -12),
        "not_nested": (8, 8),
        "unicode": (8, 8),
        "and": (8, -12),
        "negative_only": (8, 6),
    }
    for row in data:
        dx, dy = offsets.get(str(row["name"]), (6, 5))
        ax.annotate(short(str(row["name"]), 16), (int(row["l0"]), float(row["p95"])), xytext=(dx, dy), textcoords="offset points", fontsize=8)
    ax.set_xscale("log")
    ax.set_xlabel("L0 matches, log scale")
    ax.set_ylabel("p95 latency, ms")
    ax.set_title("Selectivity vs tail latency")
    finish(fig, "stress_selectivity_latency.png")


def plot_index_footprint(metrics: dict[str, str]) -> None:
    import matplotlib.pyplot as plt

    corpus = parse_mib(metrics["Corpus file size"])
    index = parse_mib(metrics["Index file size"])
    raw = int(metrics["Raw postings bytes"]) / 1024 / 1024
    compressed = int(metrics["Compressed postings bytes"]) / 1024 / 1024
    ratio = float(metrics["Compression ratio"])
    docs = int(metrics["Docs"])
    terms = int(metrics["Terms"])

    fig, axes = plt.subplots(1, 2, figsize=(11.6, 4.8))
    axes[0].bar(["corpus", "mmap index"], [corpus, index], color=["#2563eb", "#f97316"])
    axes[0].set_ylabel("MiB")
    axes[0].set_title("Corpus vs index size")
    for i, value in enumerate([corpus, index]):
        axes[0].text(i, value + 1.0, f"{value:.1f}", ha="center", fontsize=9)

    axes[1].bar(["raw postings", "compressed"], [raw, compressed], color=["#94a3b8", "#059669"])
    axes[1].set_ylabel("MiB")
    axes[1].set_title(f"Postings compression: {ratio:.3f}x")
    for i, value in enumerate([raw, compressed]):
        axes[1].text(i, value + 0.5, f"{value:.1f}", ha="center", fontsize=9)

    fig.text(0.5, -0.01, f"{docs} fragments, {terms} terms", ha="center", color="#475467")
    finish(fig, "index_footprint.png")


def plot_profile_top(profile: list[dict[str, object]]) -> None:
    import matplotlib.pyplot as plt
    import numpy as np

    data = list(reversed(profile[:10]))
    labels = [short_func(str(row["function"])) for row in data]
    y = np.arange(len(data))

    fig, ax = plt.subplots(figsize=(11.8, 6.2))
    ax.barh(y, [row["cum"] for row in data], color="#c4b5fd", label="cum")
    ax.barh(y, [row["flat"] for row in data], color="#7c3aed", label="flat")
    ax.set_yticks(y, labels)
    ax.set_xlabel("milliseconds")
    ax.set_title("CPU profile top functions")
    ax.legend(loc="lower right")
    for i, row in enumerate(data):
        ax.text(float(row["cum"]) + 15, i, f'{row["flat_pct"]:.1f}% flat', va="center", fontsize=8)
    finish(fig, "profile_cpu_top.png")


def parse_mib(value: str) -> float:
    match = re.search(r"/\s*([0-9.]+)\s*MiB", value)
    if match:
        return float(match.group(1))
    match = re.search(r"([0-9]+)", value)
    if not match:
        return 0.0
    return int(match.group(1)) / 1024 / 1024


def short(value: str, limit: int = 20) -> str:
    return value if len(value) <= limit else value[: limit - 1] + "…"


def short_func(value: str) -> str:
    value = value.replace("lab5/search.", "")
    value = value.replace("runtime.", "rt.")
    value = value.replace("unicode/", "unicode/")
    return short(value, 42)


def finish(fig, filename: str) -> None:
    import matplotlib.pyplot as plt

    fig.tight_layout()
    out = FIG / filename
    fig.savefig(out)
    plt.close(fig)
    print(out)


if __name__ == "__main__":
    main()
