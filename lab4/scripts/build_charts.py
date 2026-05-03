import os
import re
import statistics
from pathlib import Path

os.environ.setdefault("MPLCONFIGDIR", str(Path(__file__).resolve().parent.parent / ".mplconfig"))

import matplotlib

matplotlib.use("Agg")
import matplotlib.pyplot as plt

ROOT = Path(__file__).resolve().parent.parent
BENCH = ROOT / "results" / "bench.txt"
FIG = ROOT / "figures"
RES = ROOT / "results"
SIGMA = 3

LINE_RE = re.compile(
    r"^(Benchmark[^\s]+)\s+\d+\s+([0-9.]+)\s+([numµsm]+)s/op\s+([0-9.]+)\s+B/op\s+([0-9.]+)\s+allocs/op$"
)
GROWTH_RE = re.compile(r"^BenchmarkGrowth(Concurrent|Plain)(Get|Put|Merge)/N(\d+)$")
CPU_SUFFIX_RE = re.compile(r"-\d+$")


def ns_value(value: float, prefix: str) -> float:
    scale = {
        "n": 1.0,
        "u": 1_000.0,
        "µ": 1_000.0,
        "m": 1_000_000.0,
        "": 1_000_000_000.0,
    }
    return value * scale[prefix]


def parse_bench():
    groups = {}
    for raw in BENCH.read_text(encoding="utf-8").splitlines():
        line = raw.strip()
        match = LINE_RE.match(line)
        if not match:
            continue
        name = CPU_SUFFIX_RE.sub("", match.group(1))
        ns = ns_value(float(match.group(2)), match.group(3))
        b_op = float(match.group(4))
        allocs = float(match.group(5))
        groups.setdefault(name, {"ns": [], "b": [], "allocs": []})
        groups[name]["ns"].append(ns)
        groups[name]["b"].append(b_op)
        groups[name]["allocs"].append(allocs)
    return groups


def stats(values):
    mean = statistics.mean(values)
    stdev = statistics.stdev(values) if len(values) > 1 else 0.0
    return mean, SIGMA * stdev


def aggregate(groups):
    growth = {"Get": {"Concurrent": [], "Plain": []}, "Put": {"Concurrent": [], "Plain": []}, "Merge": {"Concurrent": [], "Plain": []}}
    parallel = []
    for name, values in groups.items():
        entry = {
            "name": name,
            "ns_mean": stats(values["ns"])[0],
            "ns_band": stats(values["ns"])[1],
            "b_mean": stats(values["b"])[0],
            "b_band": stats(values["b"])[1],
            "allocs_mean": stats(values["allocs"])[0],
            "allocs_band": stats(values["allocs"])[1],
        }
        match = GROWTH_RE.match(name)
        if match:
            kind, op, n = match.groups()
            entry["n"] = int(n)
            growth[op][kind].append(entry)
            continue
        if name.startswith("BenchmarkParallel"):
            parallel.append(entry)
    for op in growth.values():
        for family in op.values():
            family.sort(key=lambda row: row["n"])
    parallel.sort(key=lambda row: row["name"])
    return growth, parallel


def format_ns(ns: float) -> str:
    if ns >= 1_000_000:
        return f"{ns / 1_000_000:.2f} ms"
    if ns >= 1_000:
        return f"{ns / 1_000:.2f} µs"
    return f"{ns:.2f} ns"


def plot_growth(growth):
    FIG.mkdir(parents=True, exist_ok=True)
    colors = {"Concurrent": "#0f766e", "Plain": "#1d4ed8"}
    for op, families in growth.items():
        fig, ax = plt.subplots(figsize=(8.5, 4.8), dpi=120)
        for family, rows in families.items():
            if not rows:
                continue
            ax.plot(
                [row["n"] for row in rows],
                [row["ns_mean"] for row in rows],
                "o-",
                label=family,
                color=colors[family],
                linewidth=2,
                markersize=5,
            )
            low = [max(row["ns_mean"] - row["ns_band"], 0) for row in rows]
            high = [row["ns_mean"] + row["ns_band"] for row in rows]
            ax.fill_between([row["n"] for row in rows], low, high, color=colors[family], alpha=0.14)
        ax.set_xscale("log")
        ax.set_xlabel("N")
        ax.set_ylabel("ns/op")
        ax.set_title(f"{op}: latency vs N (±{SIGMA}σ)")
        ax.grid(True, alpha=0.25)
        ax.legend()
        fig.tight_layout()
        fig.savefig(FIG / f"growth_{op.lower()}_latency.png")
        plt.close(fig)


def plot_parallel(parallel):
    if not parallel:
        return
    FIG.mkdir(parents=True, exist_ok=True)
    names = [row["name"].replace("Benchmark", "") for row in parallel]
    ops = [1e9 / row["ns_mean"] for row in parallel]
    bands = [1e9 * row["ns_band"] / max(row["ns_mean"] ** 2, 1e-9) for row in parallel]
    fig, ax = plt.subplots(figsize=(8.5, 4.8), dpi=120)
    ax.bar(names, ops, yerr=bands, color=["#b45309", "#7c3aed"], alpha=0.86, capsize=6)
    ax.set_ylabel("ops/s")
    ax.set_title(f"Parallel workloads (±{SIGMA}σ)")
    ax.grid(True, axis="y", alpha=0.25)
    fig.tight_layout()
    fig.savefig(FIG / "parallel_throughput.png")
    plt.close(fig)


def write_tables(growth, parallel):
    RES.mkdir(parents=True, exist_ok=True)
    lines = []
    for op, families in growth.items():
        lines.append(f"## {op}\n\n")
        lines.append(f"| N | Concurrent ns/op (±{SIGMA}σ) | Plain ns/op (±{SIGMA}σ) | Concurrent B/op | Plain B/op |\n")
        lines.append("|---:|---:|---:|---:|---:|\n")
        indexed = {row["n"]: row for row in families["Concurrent"]}
        for row in families["Plain"]:
            peer = indexed.get(row["n"])
            if not peer:
                continue
            lines.append(
                f"| {row['n']} | {peer['ns_mean']:.2f} ± {peer['ns_band']:.2f} | "
                f"{row['ns_mean']:.2f} ± {row['ns_band']:.2f} | "
                f"{peer['b_mean']:.1f} | {row['b_mean']:.1f} |\n"
            )
        lines.append("\n")

    if parallel:
        lines.append("## Parallel\n\n")
        lines.append(f"| Workload | ns/op (±{SIGMA}σ) | ~ op/s | B/op |\n")
        lines.append("|---|---:|---:|---:|\n")
        for row in parallel:
            lines.append(
                f"| `{row['name'].replace('Benchmark', '')}` | {row['ns_mean']:.2f} ± {row['ns_band']:.2f} | "
                f"{1e9 / row['ns_mean']:.0f} | {row['b_mean']:.1f} |\n"
            )
        lines.append("\n")

    (RES / "tables.md").write_text("".join(lines), encoding="utf-8")


def write_summary(growth, parallel):
    RES.mkdir(parents=True, exist_ok=True)
    lines = []
    for op, families in growth.items():
        conc = families["Concurrent"][-1]
        plain = families["Plain"][-1]
        slowdown = conc["ns_mean"] / plain["ns_mean"]
        lines.append(
            f"- {op}: на N={conc['n']} concurrent = {format_ns(conc['ns_mean'])}, "
            f"plain = {format_ns(plain['ns_mean'])}, overhead = {slowdown:.2f}x"
        )
    for row in parallel:
        lines.append(f"- {row['name'].replace('Benchmark', '')}: {1e9 / row['ns_mean']:.0f} ops/s")
    (RES / "summary.md").write_text("\n".join(lines) + "\n", encoding="utf-8")


def main():
    groups = parse_bench()
    if not groups:
        raise SystemExit("bench results are empty")
    growth, parallel = aggregate(groups)
    plot_growth(growth)
    plot_parallel(parallel)
    write_tables(growth, parallel)
    write_summary(growth, parallel)


if __name__ == "__main__":
    main()
