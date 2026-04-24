#!/usr/bin/env python3
from __future__ import annotations

import json
import sys
from collections import OrderedDict
from pathlib import Path
from xml.sax.saxutils import escape


WIDTH = 1600
ROW_HEIGHT = 22
HEADER_HEIGHT = 72
FOOTER_HEIGHT = 30
LEFT_PAD = 24
RIGHT_PAD = 24
MIN_LABEL_WIDTH = 42
FONT_SIZE = 12


class Node:
    __slots__ = ("src", "value", "children")

    def __init__(self, src: int) -> None:
        self.src = src
        self.value = 0
        self.children: OrderedDict[int, Node] = OrderedDict()


def extract_stack_data(html: str) -> dict:
    marker = "stackViewer("
    start = html.rfind(marker)
    if start < 0:
        raise ValueError("stackViewer(...) not found in HTML")
    start = html.find("{", start)
    if start < 0:
        raise ValueError("stackViewer JSON start not found")

    depth = 0
    in_string = False
    escape_next = False
    end = -1
    for i in range(start, len(html)):
        ch = html[i]
        if in_string:
            if escape_next:
                escape_next = False
            elif ch == "\\":
                escape_next = True
            elif ch == '"':
                in_string = False
            continue
        if ch == '"':
            in_string = True
        elif ch == "{":
            depth += 1
        elif ch == "}":
            depth -= 1
            if depth == 0:
                end = i + 1
                break
    if end < 0:
        raise ValueError("stackViewer JSON end not found")
    return json.loads(html[start:end])


def build_tree(data: dict) -> tuple[Node, int]:
    root = Node(0)
    total = 0
    for stack in data["Stacks"]:
        value = int(stack["Value"])
        if value <= 0:
            continue
        total += value
        root.value += value
        current = root
        for src in stack["Sources"][1:]:
            child = current.children.get(src)
            if child is None:
                child = Node(src)
                current.children[src] = child
            child.value += value
            current = child
    return root, total


def max_depth(node: Node, depth: int = 0) -> int:
    best = depth
    for child in node.children.values():
        best = max(best, max_depth(child, depth + 1))
    return best


def fmt_unit(raw_value: int, scale: float, unit: str) -> str:
    value = raw_value * scale
    if unit == "s":
        if value >= 1:
            return f"{value:.2f}s"
        if value >= 1e-3:
            return f"{value * 1e3:.1f}ms"
        if value >= 1e-6:
            return f"{value * 1e6:.1f}us"
        return f"{value * 1e9:.1f}ns"
    if unit == "B":
        if value >= 1024 ** 3:
            return f"{value / 1024 ** 3:.2f}GiB"
        if value >= 1024 ** 2:
            return f"{value / 1024 ** 2:.2f}MiB"
        if value >= 1024:
            return f"{value / 1024:.1f}KiB"
        return f"{value:.0f}B"
    return f"{value:.2f}{unit}"


def color_for(full_name: str, index: int) -> str:
    digest = index
    for ch in full_name:
        digest = ((digest * 131) + ord(ch)) & 0xFFFFFFFF
    hue = 8 + digest % 52
    sat = 72 + (digest >> 8) % 18
    light = 58 + (digest >> 16) % 16
    return f"hsl({hue},{sat}%,{light}%)"


def node_meta(node: Node, sources: list[dict]) -> tuple[str, list[str], str]:
    if node.src == 0:
        return "program", ["program"], "hsl(38,82%,62%)"
    src = sources[node.src]
    return src["FullName"], src.get("Display") or [src["FullName"]], color_for(src["FullName"], node.src)


def pick_label(candidates: list[str], width: float) -> str:
    for label in candidates:
        if len(label) * FONT_SIZE * 0.58 <= width - 8:
            return label
    if width < MIN_LABEL_WIDTH:
        return ""
    fallback = candidates[-1]
    max_chars = max(3, int((width - 8) / (FONT_SIZE * 0.58)))
    if len(fallback) <= max_chars:
        return fallback
    return fallback[: max(1, max_chars - 1)] + "…"


def find_focus_node(root: Node, sources: list[dict], prefix: str) -> Node | None:
    best: tuple[int, int, int, Node] | None = None

    def visit(node: Node, depth: int) -> None:
        nonlocal best
        if node.src != 0:
            full_name = sources[node.src]["FullName"]
            if full_name.startswith(prefix):
                candidate = (-depth, node.value, -node.src, node)
                if best is None or candidate > best:
                    best = candidate
        for child in node.children.values():
            visit(child, depth + 1)

    visit(root, 0)
    return None if best is None else best[3]


def render_svg(data: dict, title: str, focus_prefix: str | None = None) -> str:
    root, total = build_tree(data)
    if total == 0:
        raise ValueError("no positive stacks found")

    sources = data["Sources"]
    subtitle_suffix = ""
    if focus_prefix:
        focus_node = find_focus_node(root, sources, focus_prefix)
        if focus_node is not None:
            root = focus_node
            total = root.value
            subtitle_suffix = f" • focused on {focus_prefix}"
    depth = max_depth(root)
    chart_width = WIDTH - LEFT_PAD - RIGHT_PAD
    height = HEADER_HEIGHT + (depth + 1) * ROW_HEIGHT + FOOTER_HEIGHT
    scale = chart_width / total

    parts: list[str] = []
    parts.append(
        f'<svg xmlns="http://www.w3.org/2000/svg" width="{WIDTH}" height="{height}" '
        f'viewBox="0 0 {WIDTH} {height}" role="img" aria-label="{escape(title)}">'
    )
    parts.append(
        "<style>"
        "text{font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Helvetica,Arial,sans-serif;}"
        ".title{font-size:24px;font-weight:700;fill:#222;}"
        ".subtitle{font-size:13px;fill:#5d5d5d;}"
        ".box-label{font-size:12px;fill:#222;pointer-events:none;}"
        ".box{stroke:#ffffff;stroke-width:1;}"
        "</style>"
    )
    parts.append(f'<rect x="0" y="0" width="{WIDTH}" height="{height}" fill="#fffaf5"/>')
    parts.append(f'<text class="title" x="{LEFT_PAD}" y="34">{escape(title)}</text>')
    summary = (
        f"pprof static flamegraph • total samples {fmt_unit(total, data['Scale'], data['Unit'])} "
        f"• widest path is the hottest path{subtitle_suffix}"
    )
    parts.append(f'<text class="subtitle" x="{LEFT_PAD}" y="56">{escape(summary)}</text>')

    def draw(node: Node, x: float, level: int) -> None:
        y = HEADER_HEIGHT + (depth - level) * ROW_HEIGHT
        width = node.value * scale
        if width <= 0:
            return
        full_name, candidates, fill = node_meta(node, sources)
        percent = node.value / total * 100
        parts.append(
            f'<g><title>{escape(full_name)}&#10;'
            f'{escape(fmt_unit(node.value, data["Scale"], data["Unit"]))} '
            f'({percent:.1f}%)</title>'
        )
        parts.append(
            f'<rect class="box" x="{x:.2f}" y="{y:.2f}" width="{width:.2f}" '
            f'height="{ROW_HEIGHT - 2}" rx="3" ry="3" fill="{fill}"/>'
        )
        label = pick_label(candidates, width)
        if label:
            parts.append(
                f'<text class="box-label" x="{x + 4:.2f}" y="{y + 14:.2f}">{escape(label)}</text>'
            )
        parts.append("</g>")

        offset = x
        for child in node.children.values():
            child_width = child.value * scale
            if child_width <= 0:
                continue
            draw(child, offset, level + 1)
            offset += child_width

    draw(root, LEFT_PAD, 0)
    parts.append(
        f'<text class="subtitle" x="{LEFT_PAD}" y="{height - 10}">'
        "Generated from go tool pprof flamegraph data"
        "</text>"
    )
    parts.append("</svg>")
    return "\n".join(parts)


def main(argv: list[str]) -> int:
    if len(argv) < 3 or len(argv) > 5:
        print(
            "usage: render_pprof_flame_svg.py <input.html> <output.svg> [title] [focus_prefix]",
            file=sys.stderr,
        )
        return 2

    input_path = Path(argv[1])
    output_path = Path(argv[2])
    title = argv[3] if len(argv) >= 4 else output_path.stem.replace("_", " ")
    focus_prefix = argv[4] if len(argv) == 5 else None

    html = input_path.read_text(encoding="utf-8")
    data = extract_stack_data(html)
    svg = render_svg(data, title, focus_prefix)
    output_path.write_text(svg, encoding="utf-8")
    print(output_path)
    return 0


if __name__ == "__main__":
    raise SystemExit(main(sys.argv))
