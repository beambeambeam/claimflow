"""
One check for the whole report: what is still missing, and what looks wrong.

    python scripts/check_report.py            # after building main.pdf
    python scripts/check_report.py --quick    # skip the pixel scans

Reports, in this order:

  1. pending figures: every \\evidence-family figure whose file does not exist;
  2. \\pending boxes (evidence that waits for a ticket or the release);
  3. \\todo markers, grouped by what they wait for. Write each as
     \\todo{release: ...}, \\todo{1045: ...}, \\todo{#30: ...} or
     \\todo{board: ...}; an unprefixed one is listed as "other";
  4. captures with blank bands or repeated content (pixel scans of
     figures/docs and figures/github; pieces are crops of those and skipped);
  5. Mermaid diagrams that did not draw, from the sidecar files;
  6. overfull boxes in main.log, with the nearest page.

The final PDF needs all of 1-5 empty. Exit status is 1 while anything remains.
"""

import json
import re
import sys
from collections import defaultdict
from pathlib import Path

BASE = Path(__file__).resolve().parent.parent
sys.path.insert(0, str(BASE / "scripts"))

CITE = re.compile(r"\\evidence(?:part|half|small)?\{([^}]+)\}")
TODO = re.compile(r"\\todo\{((?:[^{}]|\{[^{}]*\})*)\}")
PENDING = re.compile(r"\\pending\{((?:[^{}]|\{[^{}]*\})*)\}\{((?:[^{}]|\{[^{}]*\})*)\}")
PIECE = re.compile(r"-p\d+$")


def tex_files():
    files = [BASE / "main.tex"] + sorted((BASE / "content").glob("*.tex")) \
        + sorted((BASE / "content" / "data").glob("*.tex"))
    return [f for f in files if f.exists()]


def group_of(text):
    m = re.match(r"\s*(release|board|1045|capture|#\d+|ticket)\b", text)
    return m.group(1) if m else "other"


def main():
    quick = "--quick" in sys.argv
    problems = 0

    # 1-3: sources
    missing, pendings, todos = [], [], defaultdict(list)
    for f in tex_files():
        text = f.read_text(encoding="utf-8")
        for m in CITE.finditer(text):
            path = m.group(1)
            if not (BASE / path).is_file():
                missing.append((path, f.name))
        for m in PENDING.finditer(text):
            pendings.append((m.group(1), m.group(2), f.name))
        for m in TODO.finditer(text):
            todos[group_of(m.group(1))].append((m.group(1), f.name))

    print(f"1. Pending figures: {len(missing)}")
    for path, where in missing:
        print(f"   {path}   ({where})")
    print(f"2. Pending boxes: {len(pendings)}")
    for what, waits, where in pendings:
        print(f"   {where}: {what}  -> waits for: {waits}")
    n_todo = sum(len(v) for v in todos.values())
    print(f"3. TODO markers: {n_todo}")
    for group in sorted(todos):
        print(f"   [{group}] {len(todos[group])}")
        for text, where in todos[group]:
            print(f"      {where}: {text[:110]}")
    problems += len(missing) + len(pendings) + n_todo

    # 4: captures
    if quick:
        print("4. Captures: skipped (--quick)")
    else:
        from check_blank import blank_runs
        from check_repeat import repeats
        bad = 0
        total = 0
        for folder in (BASE / "figures" / "docs", BASE / "figures" / "github"):
            for p in sorted(folder.glob("*.png")) if folder.is_dir() else []:
                if PIECE.search(p.stem) or p.stem.startswith("mermaid-"):
                    continue
                total += 1
                holes, doubled = blank_runs(p), repeats(p)
                if holes or doubled:
                    bad += 1
                    print(f"   {p.relative_to(BASE)}: {len(holes)} blank band(s), "
                          f"{len(doubled)} repeated block(s)")
        print(f"4. Captures with blank bands or repeated content: {bad} of {total}")
        problems += bad

    # 5: mermaid
    undrawn = []
    for j in sorted((BASE / "figures" / "docs").glob("*.json")):
        meta = json.loads(j.read_text(encoding="utf-8"))
        if meta.get("mermaid_drawn", 0) < meta.get("mermaid_expected", 0):
            undrawn.append(f"{meta['file']}: {meta['mermaid_drawn']}/{meta['mermaid_expected']}")
    expected = sum(json.loads(j.read_text(encoding="utf-8")).get("mermaid_expected", 0)
                   for j in (BASE / "figures" / "docs").glob("*.json"))
    print(f"5. Mermaid diagrams not drawn: {len(undrawn)} (of {expected} expected in captured documents)")
    for u in undrawn:
        print(f"   {u}")
    problems += len(undrawn)

    # 6: log
    log = BASE / "main.log"
    if not log.exists():
        print("6. Overfull boxes: main.log not found (build first)")
    else:
        page, over = 1, []
        for line in log.read_text(encoding="utf-8", errors="replace").splitlines():
            for m in re.finditer(r"\[(\d+)[\s\]<{]", line):
                page = int(m.group(1))
            if line.startswith("Overfull"):
                over.append((page, line.strip()[:110]))
        print(f"6. Overfull boxes in main.log: {len(over)}")
        for pg, line in over[:25]:
            print(f"   near page {pg}: {line}")
        problems += len(over)

    print("\nclean" if problems == 0 else f"\n{problems} item(s) remain")
    sys.exit(1 if problems else 0)


if __name__ == "__main__":
    main()
