#!/usr/bin/env python3
import re
import subprocess
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
FIG = ROOT / "figures"
RESULTS = ROOT / "results"

LINE = re.compile(
    r"^(Benchmark\S+)\s+(\d+)\s+(\d+)\s+ns/op\s+(\d+)\s+B/op\s+(\d+)\s+allocs/op"
)


def run_bench():
    RESULTS.mkdir(parents=True, exist_ok=True)
    out = RESULTS / "bench_raw.txt"
    cmd = [
        "go",
        "test",
        "./hashtable",
        "./perfecthash",
        "./lsh",
        "-run=^$",
        "-bench=.",
        "-benchmem",
        "-benchtime=2s",
        "-count=1",
    ]
    p = subprocess.run(cmd, cwd=ROOT, capture_output=True, text=True)
    out.write_text(p.stdout + p.stderr, encoding="utf-8")
    if p.returncode != 0:
        print(p.stderr, file=sys.stderr)
        sys.exit(p.returncode)
    return out.read_text(encoding="utf-8")


def parse(text):
    rows = []
    for line in text.splitlines():
        m = LINE.match(line.strip())
        if m:
            raw = re.sub(r"-\d+$", "", m.group(1))
            rows.append({"name": raw, "ns": int(m.group(3)), "b": int(m.group(4))})
    return rows


def svg_bar_chart(title, xlabels, series, path, ylabel):
    w, h = 760, 440
    margin_l, margin_r, margin_t, margin_b = 72, 200, 52, 72
    plot_w = w - margin_l - margin_r
    plot_h = h - margin_t - margin_b
    n = len(xlabels)
    ng = len(series)
    group_w = plot_w / max(n, 1)
    bar_w = group_w / (ng + 1)
    maxv = max(max(vals) for _, vals in series) or 1

    parts = [
        f'<svg xmlns="http://www.w3.org/2000/svg" width="{w}" height="{h}" '
        f'font-family="system-ui,Segoe UI,sans-serif" font-size="13">',
        '<rect width="100%" height="100%" fill="#fafafa"/>',
        f'<text x="{margin_l}" y="34" font-size="17" font-weight="600">{title}</text>',
        f'<text x="{margin_l}" y="{h - 22}" fill="#666" font-size="11">'
        f"Данные: go test ./hashtable ./perfecthash ./lsh -run=^$ -bench=. -benchmem -benchtime=2s</text>",
    ]
    colors = ["#2563eb", "#059669", "#d97706", "#7c3aed"]
    for gi, (gname, vals) in enumerate(series):
        for i, v in enumerate(vals):
            gx = margin_l + i * group_w + bar_w * 0.5 + gi * bar_w
            bh = (v / maxv) * plot_h
            y = margin_t + plot_h - bh
            parts.append(
                f'<rect x="{gx:.1f}" y="{y:.1f}" width="{bar_w * 0.85:.1f}" height="{max(bh, 1):.1f}" '
                f'fill="{colors[gi % len(colors)]}" rx="3"/>'
            )
    for i, lab in enumerate(xlabels):
        cx = margin_l + i * group_w + group_w / 2
        parts.append(
            f'<text x="{cx:.0f}" y="{margin_t + plot_h + 28}" text-anchor="middle" '
            f'fill="#222" font-size="13">{lab}</text>'
        )
    parts.append(
        f'<text transform="translate(22,{margin_t + plot_h / 2}) rotate(-90)" '
        f'text-anchor="middle" fill="#333">{ylabel}</text>'
    )
    ly = margin_t + 8
    for gi, (gname, _) in enumerate(series):
        parts.append(
            f'<rect x="{w - margin_r + 10}" y="{ly + gi * 22}" width="14" height="14" '
            f'fill="{colors[gi % len(colors)]}" rx="2"/>'
        )
        parts.append(
            f'<text x="{w - margin_r + 32}" y="{ly + 12 + gi * 22}" fill="#222">{gname}</text>'
        )
    parts.append("</svg>")
    path.write_text("\n".join(parts), encoding="utf-8")


def main():
    FIG.mkdir(parents=True, exist_ok=True)
    raw = run_bench()
    rows = parse(raw)
    if not rows:
        print("no benchmark lines parsed", file=sys.stderr)
        sys.exit(1)
    m = {r["name"]: r["ns"] for r in rows}

    def g(name):
        return m.get(name, 0)

    x_sml = ["S", "M", "L"]
    svg_bar_chart(
        "Файловая хэш-таблица: Set / Get / Delete (µs/op)",
        x_sml,
        [
            ("Set", [g(f"BenchmarkSet_{s}") / 1000 for s in ("Small", "Medium", "Large")]),
            ("Get", [g(f"BenchmarkGet_{s}") / 1000 for s in ("Small", "Medium", "Large")]),
            ("Delete", [g(f"BenchmarkDelete_{s}") / 1000 for s in ("Small", "Medium", "Large")]),
        ],
        FIG / "disk_hashtable_ops.svg",
        "µs/op",
    )

    xk = ["1K", "10K", "50K"]
    svg_bar_chart(
        "Perfect hash: Build (ms/op)",
        xk,
        [("Build", [g(f"BenchmarkBuild_{k}") / 1e6 for k in ("1K", "10K", "50K")])],
        FIG / "perfecthash_build.svg",
        "ms/op",
    )
    svg_bar_chart(
        "Perfect hash: Get (ns/op)",
        xk,
        [("Get", [g(f"BenchmarkGet_{k}") for k in ("1K", "10K", "50K")])],
        FIG / "perfecthash_get.svg",
        "ns/op",
    )

    xlm = ["Small", "Medium", "Large"]
    svg_bar_chart(
        "LSH vs наивный перебор: FindDuplicates (ms/op)",
        xlm,
        [
            ("LSH", [g(f"BenchmarkFindDuplicates_{x}") / 1e6 for x in xlm]),
            ("Naive", [g(f"BenchmarkNaiveFindDuplicates_{x}") / 1e6 for x in xlm]),
        ],
        FIG / "lsh_vs_naive.svg",
        "ms/op",
    )

    for p in sorted(FIG.glob("*.svg")):
        print("wrote", p)


if __name__ == "__main__":
    main()
