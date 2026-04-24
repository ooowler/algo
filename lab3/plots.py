import json
import os
from pathlib import Path

os.environ.setdefault("MPLCONFIGDIR", str(Path(__file__).resolve().parent / ".mplconfig"))
os.environ.setdefault("XDG_CACHE_HOME", str(Path(__file__).resolve().parent / ".cache"))

import matplotlib

matplotlib.use("Agg")
import matplotlib.pyplot as plt

FIG = Path(__file__).resolve().parent / "figures"
RES = Path(__file__).resolve().parent / "results"


def split_rows(rows):
    d = {"LSH": [], "HNSW": [], "IVFPQ": []}
    for r in rows:
        d[r["name"].split()[0]].append(r)
    return d


def best_of(rows):
    if not rows:
        return None
    return max(rows, key=lambda r: (r["recall"], -r["search_ms_per_q"], -r["index_bytes"]))


def score(r):
    return r["recall"] / (1e-6 + r["search_ms_per_q"]) ** 0.25 / (1e-6 + r["index_bytes"]) ** 0.1 / (
        1e-6 + r["build_s"]
    ) ** 0.05


def plot_all(rows, title_prefix, bench_count, sigma):
    FIG.mkdir(parents=True, exist_ok=True)
    by = split_rows(rows)
    colors = {"LSH": "#2563eb", "HNSW": "#059669", "IVFPQ": "#d97706"}
    labels = {"LSH": "LSH", "HNSW": "HNSW", "IVFPQ": "IVF+PQ"}

    def scatter(ax, xk, yk, xlab, ylab, title, logx=False, logy=False):
        for k, rs in by.items():
            if not rs:
                continue
            ax.scatter(
                [r[xk] for r in rs],
                [r[yk] for r in rs],
                s=32,
                label=labels[k],
                color=colors[k],
                alpha=0.78,
                edgecolors="white",
                linewidths=0.4,
            )
        ax.set_xlabel(xlab)
        ax.set_ylabel(ylab)
        ax.set_title(title_prefix + title)
        if logx:
            ax.set_xscale("log")
        if logy:
            ax.set_yscale("log")
        ax.grid(True, alpha=0.3)
        ax.legend(loc="best")

    w, h, dpi = 10, 5, 120
    fig, ax = plt.subplots(figsize=(w, h), dpi=dpi)
    scatter(
        ax,
        "search_ms_per_q",
        "recall",
        "Задержка на запрос, мс (log)",
        "Recall@100",
        "recall vs задержка поиска",
        logx=True,
    )
    fig.tight_layout()
    fig.savefig(FIG / "ann_recall_latency.png")
    plt.close(fig)

    fig, ax = plt.subplots(figsize=(w, h), dpi=dpi)
    scatter(
        ax,
        "index_bytes",
        "recall",
        "Размер индекса, байт (log)",
        "Recall@100",
        "recall vs размер индекса",
        logx=True,
    )
    fig.tight_layout()
    fig.savefig(FIG / "ann_recall_size.png")
    plt.close(fig)

    fig, ax = plt.subplots(figsize=(w, h), dpi=dpi)
    scatter(
        ax,
        "build_s",
        "recall",
        "Время построения, с (log)",
        "Recall@100",
        "recall vs время индексации",
        logx=True,
    )
    fig.tight_layout()
    fig.savefig(FIG / "ann_recall_build.png")
    plt.close(fig)

    fig, ax = plt.subplots(figsize=(w, h), dpi=dpi)
    scatter(
        ax,
        "search_ms_per_q",
        "index_bytes",
        "Задержка на запрос, мс (log)",
        "Размер индекса, байт (log)",
        "размер индекса vs задержка",
        logx=True,
        logy=True,
    )
    fig.tight_layout()
    fig.savefig(FIG / "ann_size_latency.png")
    plt.close(fig)

    fig, ax = plt.subplots(figsize=(w, h), dpi=dpi)
    for k, rs in by.items():
        if not rs:
            continue
        rs = sorted(rs, key=lambda r: r["recall"])
        ax.plot(
            [r["recall"] for r in rs],
            [r["search_ms_per_q"] for r in rs],
            "o-",
            linewidth=2,
            markersize=5,
            label=labels[k],
            color=colors[k],
        )
    ax.set_xlabel("Recall@100")
    ax.set_ylabel("Задержка на запрос, мс")
    ax.set_yscale("log")
    ax.set_title(title_prefix + "Pareto-подобная кривая (параметры внутри семейства)")
    ax.grid(True, alpha=0.3)
    ax.legend()
    fig.tight_layout()
    fig.savefig(FIG / "ann_recall_latency_lines.png")
    plt.close(fig)

    fig, ax = plt.subplots(figsize=(w, h), dpi=dpi)
    for k, rs in by.items():
        if not rs:
            continue
        xm = [r["search_ms_per_q"] for r in rs]
        ym = [r["recall"] for r in rs]
        sm = [r.get("search_ms_stdev", 0) or 0 for r in rs]
        xlow = [max(m - sigma * s, 0) for m, s in zip(xm, sm)]
        xhigh = [m + sigma * s for m, s in zip(xm, sm)]
        xel = [m - lo for m, lo in zip(xm, xlow)]
        xer = [hi - m for m, hi in zip(xm, xhigh)]
        ax.errorbar(
            xm,
            ym,
            xerr=[xel, xer],
            fmt="o",
            color=colors[k],
            label=labels[k],
            capsize=4,
            markersize=7,
            alpha=0.88,
            elinewidth=1.2,
        )
    ax.set_xlabel(
        f"Задержка на запрос, мс (среднее; полоса ±{sigma}σ по времени search, {bench_count} прогонов)"
    )
    ax.set_ylabel("Recall@100")
    ax.set_title(title_prefix + f"recall vs latency (±{sigma}σ по поиску)")
    ax.grid(True, alpha=0.3)
    ax.legend(loc="best")
    fig.tight_layout()
    fig.savefig(FIG / "ann_recall_latency_3sigma.png")
    plt.close(fig)


def write_tables(rows, sigma):
    RES.mkdir(parents=True, exist_ok=True)
    lines = []
    for fam in ("LSH", "HNSW", "IVFPQ"):
        rs = [r for r in rows if r["name"].startswith(fam)]
        rs.sort(key=lambda r: (r["recall"], r["search_ms_per_q"]))
        lines.append(f"## {fam}\n\n")
        lines.append(
            f"| Конфигурация | Recall@100 | Построение, с | Поиск, мс/q (±{sigma}σ) | Размер, МБ |\n"
        )
        lines.append("|---|---:|---:|---:|---:|\n")
        for r in rs:
            s = r.get("search_ms_stdev") or 0
            lines.append(
                f"| `{r['name']}` | {r['recall']:.4f} | {r['build_s']:.2f} | "
                f"{r['search_ms_per_q']:.4f} ± {sigma * s:.4f} | {r['index_bytes'] / 1e6:.2f} |\n"
            )
        lines.append("\n")
    (RES / "tables.md").write_text("".join(lines), encoding="utf-8")


def best_line(b, sigma):
    if not b:
        return "—"
    return (
        f"- `{b['name']}`\n"
        f"- recall@100: **{b['recall']:.4f}**\n"
        f"- поиск: **{b['search_ms_per_q']:.4f} ± {sigma * (b.get('search_ms_stdev') or 0):.4f}** мс/q\n"
        f"- построение: **{b['build_s']:.2f}** с\n"
        f"- размер: **{b['index_bytes'] / 1e6:.2f}** МБ\n"
    )


def write_report(rows, demo, sigma):
    RES.mkdir(parents=True, exist_ok=True)
    payload = {"demo": demo, "rows": rows}
    (RES / "metrics.json").write_text(json.dumps(payload, indent=2), encoding="utf-8")
    by = split_rows(rows)
    lines = []
    if demo:
        lines.append("*(режим LAB3_DEMO=1: случайные векторы, не SIFT1M)*\n\n")
    for k in ("LSH", "HNSW", "IVFPQ"):
        b = best_of(by[k])
        lines.append(f"## {k}\n\n{best_line(b, sigma)}\n")
    overall = max(rows, key=score)
    lines.append(f"## Итог (эвристический компромисс recall / latency / размер / build)\n\n{best_line(overall, sigma)}\n")
    (RES / "report.md").write_text("\n".join(lines), encoding="utf-8")


def export_all(rows, demo, title_prefix, bench_count, sigma):
    plot_all(rows, title_prefix, bench_count, sigma)
    write_tables(rows, sigma)
    write_report(rows, demo, sigma)
