# Copyright (c) 2026 Michael Lechner
# Licensed under the MIT License. See LICENSE file in the project root for full license information.
"""Turn real gitpulse terminal output (with ANSI colours) into a terminal-window
SVG in the style of the mlcprodweb pages.

    ansi2svg.py TITLE OUT.svg [--max-width N] [--drop-repos a,b] [--no-summary] < output

Lines starting with "$ " are drawn as a prompt and a bold command.
--drop-repos leaves out the table row and -v block of the named repositories,
--no-summary stops before the SUMMARY block, --max-width cuts long lines.
"""
import argparse
import re
import sys
from html import escape

CW, LH, X0, Y0 = 8.4, 21, 24.0, 64
FG = {"90": "#9ca3af", "91": "#f87171", "92": "#4ade80", "93": "#fb923c", "94": "#38bdf8",
      "95": "#f472b6", "96": "#22d3ee", None: "#d4d4d8"}
SGR = re.compile(r"\x1b\[([0-9;]*)m")

def parse(line):
    """Split a line into (text, color, bold, underline) runs."""
    runs, st = [], {"fg": None, "b": False, "u": False}
    pos = 0
    for m in SGR.finditer(line):
        if m.start() > pos:
            runs.append((line[pos:m.start()], dict(st)))
        for code in (m.group(1) or "0").split(";"):
            if code in ("0", ""): st = {"fg": None, "b": False, "u": False}
            elif code == "1": st["b"] = True
            elif code == "4": st["u"] = True
            elif code in FG: st["fg"] = code
        pos = m.end()
    if pos < len(line):
        runs.append((line[pos:], dict(st)))
    merged = []
    for text, s in runs:
        if merged and merged[-1][1] == s:
            merged[-1] = (merged[-1][0] + text, s)
        else:
            merged.append((text, s))
    return merged

def render(title, lines, out, maxw=None):
    parsed = [parse(l) for l in lines]
    width = max(sum(len(t) for t, _ in p) for p in parsed if p)
    if maxw: width = min(width, maxw)
    w = round(width * CW + 2 * X0); h = Y0 + (len(lines) - 1) * LH + 22
    o = [f'<svg xmlns="http://www.w3.org/2000/svg" width="{w}" height="{h}" viewBox="0 0 {w} {h}" role="img" aria-label="{escape(title)}">',
         f'<rect width="{w}" height="{h}" rx="10" fill="#18181b"/>',
         f'<rect width="{w}" height="36" rx="10" fill="#27272a"/><rect y="26" width="{w}" height="10" fill="#27272a"/>',
         '<circle cx="20" cy="18" r="6" fill="#ef4444"/><circle cx="40" cy="18" r="6" fill="#eab308"/><circle cx="60" cy="18" r="6" fill="#22c55e"/>',
         f'<text x="{w/2:.1f}" y="23" text-anchor="middle" font-family="ui-sans-serif,system-ui,sans-serif" font-size="13" fill="#71717a">{escape(title)}</text>',
         '<g font-family="ui-monospace,SFMono-Regular,Menlo,Consolas,monospace" font-size="14" xml:space="preserve">']
    for i, runs in enumerate(parsed):
        x, y, used = X0, Y0 + i * LH, 0
        for text, s in runs:
            if maxw and used + len(text) > maxw:
                text = text[:max(0, maxw - used)]
            if text.strip():
                attrs = (' font-weight="bold"' if s["b"] else '') + (' text-decoration="underline"' if s["u"] else '')
                fill = "#fafafa" if (s["b"] and s["fg"] is None) else FG.get(s["fg"], FG[None])
                o.append(f'<text x="{x:.1f}" y="{y}" fill="{fill}"{attrs}>{escape(text)}</text>')
            x += len(text) * CW; used += len(text)
    o.append('</g></svg>')
    open(out, "w").write("\n".join(o) + "\n")
    print(out, w, h)

PLAIN = re.compile(r"\x1b\[[0-9;]*m")


def plain(line):
    return PLAIN.sub("", line)


def trim(lines, drop, no_summary):
    """Drop whole repository blocks and, optionally, the summary."""
    keep, skipping = [], False
    for line in lines:
        p = plain(line)
        if no_summary and p.startswith("SUMMARY"):
            break
        if p and not p.startswith((" ", "─", "$")):
            skipping = p.split(" ")[0] in drop
        if not skipping:
            keep.append(line)
    while keep and plain(keep[-1]).strip().strip("─") == "":
        keep.pop()
    return keep


def main():
    ap = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    ap.add_argument("title")
    ap.add_argument("out")
    ap.add_argument("--max-width", type=int)
    ap.add_argument("--drop-repos", default="")
    ap.add_argument("--no-summary", action="store_true")
    a = ap.parse_args()
    lines = sys.stdin.read().rstrip("\n").split("\n")
    lines = trim(lines, {d for d in a.drop_repos.split(",") if d}, a.no_summary)
    lines = [("\x1b[95m$ \x1b[0m\x1b[1m" + l[2:] + "\x1b[0m") if l.startswith("$ ") else l for l in lines]
    render(a.title, lines, a.out, a.max_width)


if __name__ == "__main__":
    main()
