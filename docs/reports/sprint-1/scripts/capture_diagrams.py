"""
Capture every Mermaid diagram of the cited documents at full size, for print.

render_docs.py photographs a whole document at 1x, so a wide diagram shrinks
to the page width with it and its labels become unreadable. This script opens
the same GitHub file pages, waits for each diagram to draw, and photographs
each diagram block on its own at three times the pixel density. The report
prints these after the document they belong to, one diagram per figure.

    python scripts/capture_diagrams.py --ref ac5bce5 --label main

Output: figures/diagrams/<name>-<n>.png and figures/diagrams/index.json
(file, ref, commit, and the diagrams in source order).
"""

import json
import sys
from pathlib import Path

from PIL import Image, ImageOps
from playwright.sync_api import sync_playwright

from render_docs import (DOCS, FILE_VIEW, REPO, commit_at, hide_diagram_controls,
                         source_at, wait_for_mermaid)

HERE = Path(__file__).resolve().parent
OUT = HERE.parent / "figures" / "diagrams"
SCALE = 3
WIDTH = 1400


def trim(target, pad=24):
    """Cut the white margin GitHub leaves around a centred diagram."""
    img = Image.open(target).convert("RGB")
    ink = ImageOps.invert(img.convert("L")).point(lambda v: 255 if v > 12 else 0)
    box = ink.getbbox()
    if box:
        l, t, r, b = box
        img.crop((max(l - pad, 0), max(t - pad, 0),
                  min(r + pad, img.width), min(b + pad, img.height))).save(target)


def capture(page, name, path, ref):
    expected = source_at(ref, path).count("```mermaid")
    if not expected:
        return []
    page.goto(f"{REPO}/blob/{ref}/{path}", wait_until="domcontentloaded", timeout=60_000)
    for sel in FILE_VIEW:
        page.wait_for_selector(sel, timeout=30_000)
    page.wait_for_timeout(1500)
    _, drawn, errors = wait_for_mermaid(page, expected)
    hide_diagram_controls(page)
    blocks = page.locator('section[data-type="mermaid"], section.js-render-needs-enrichment')
    shots = []
    for n in range(blocks.count()):
        block = blocks.nth(n)
        block.scroll_into_view_if_needed()
        page.wait_for_timeout(1200)
        target = OUT / f"{name}-{n + 1}.png"
        block.screenshot(path=str(target))
        trim(target)
        shots.append(target.name)
    status = f"{len(shots)}/{expected} captured, {drawn} drawn"
    if errors:
        status += f", errors: {errors}"
    print(f"  {name:<14} {status}")
    return shots


def main():
    args = sys.argv[1:]
    ref, label = "main", None
    if "--ref" in args:
        i = args.index("--ref")
        ref = args[i + 1]
        del args[i:i + 2]
    if "--label" in args:
        i = args.index("--label")
        label = args[i + 1]
        del args[i:i + 2]
    OUT.mkdir(parents=True, exist_ok=True)
    index = {"ref": label or ref, "commit": commit_at(ref), "docs": {}}
    with sync_playwright() as p:
        try:
            browser = p.chromium.launch()
        except Exception:
            browser = p.chromium.launch(channel="chrome")
        page = browser.new_page(viewport={"width": WIDTH, "height": 1100},
                                device_scale_factor=SCALE)
        for name, path in DOCS.items():
            if args and name not in args:
                continue
            shots = capture(page, name, path, ref)
            if shots:
                index["docs"][name] = {"file": path, "diagrams": shots}
        browser.close()
    (OUT / "index.json").write_text(json.dumps(index, indent=2), encoding="utf-8")


if __name__ == "__main__":
    main()
