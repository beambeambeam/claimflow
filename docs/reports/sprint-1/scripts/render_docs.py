"""
Capture the repository documents the report cites, as GitHub renders them.

Parts A-E of the report are the repo documents themselves. GitHub already
renders them (tables, Mermaid diagrams, task lists), so this drives a headless
browser over each file page and photographs the rendered file box. The
repository is public, so no login is needed.

    python scripts/render_docs.py --ref sprint-1            # every document
    python scripts/render_docs.py --ref main srs plan       # just those

Output: figures/docs/<name>.png plus figures/docs/<name>.json, a sidecar that
records the file, the ref, the commit the ref pointed at, and how many Mermaid
diagrams the source holds against how many had drawn. The report reads the
sidecar to print "captured from <file> at <ref>".

Ported from the solo-lab pipeline: the scroll walk (GitHub paints long markdown
lazily), the banded stitch past Skia's 16384 px texture limit, and the blank
and repeat checks after every capture. New here: Mermaid. GitHub draws each
diagram inside an iframe after the page loads, so the script waits until every
diagram has a height, then scrolls each one into view so the iframe paints.
"""

import io
import json
import re
import subprocess
import sys
import urllib.request
from pathlib import Path

from PIL import Image

Image.MAX_IMAGE_PIXELS = None
from playwright.sync_api import sync_playwright

from check_blank import blank_runs
from check_repeat import repeats

PAUSES = (150, 400, 900)
MAX_TEXTURE = 16384
PAD = 24
# CSS px. At 1400 a 14 px line is about 6 pt on the page; below GitHub's 1012 px
# breakpoint the sidebar goes and the file box fills the width, so text prints
# near 8 pt.
WIDTH = 800

REPO = "https://github.com/beambeambeam/claimflow"
HERE = Path(__file__).resolve().parent
OUT = HERE.parent / "figures" / "docs"

# The breadcrumb carries the file path, the box below carries the content, so
# the capture proves which file is on screen.
FILE_VIEW = ['[data-testid="breadcrumbs"]', "div[data-hpc]"]

# name -> repo path. The order is the order of the report.
DOCS = {
    "srs":          "docs/srs.md",
    "features":     "docs/features.md",
    "traceability": "docs/traceability.md",
    "system-sds":   "docs/system-sds.md",
    "adr-0001":     "docs/adr/0001-finance-officer-reviews-department-head.md",
    "plan":         "docs/sprints/plan.md",
    "f01":          "docs/sprints/sprint-1/F-01-authentication.md",
    "f02":          "docs/sprints/sprint-1/F-02-submit-claim.md",
    "f03":          "docs/sprints/sprint-1/F-03-my-claims.md",
    "f04":          "docs/sprints/sprint-1/F-04-review-decision.md",
    "dod":          "docs/sprints/sprint-1/definition-of-done.md",
}


def launch(p):
    """Chrome if installed, else Edge. Neither needs a download."""
    for channel in ("chrome", "msedge"):
        try:
            return p.chromium.launch(channel=channel, headless=True)
        except Exception:
            continue
    sys.exit("neither Chrome nor Edge could be launched")


def source_at(ref, path):
    """The raw markdown at the ref, to count what the page should show."""
    url = f"https://raw.githubusercontent.com/beambeambeam/claimflow/{ref}/{path}"
    with urllib.request.urlopen(url, timeout=30) as r:
        return r.read().decode("utf-8")


def commit_at(ref):
    """The commit the ref points at on GitHub, via the public API."""
    url = f"https://api.github.com/repos/beambeambeam/claimflow/commits/{ref}"
    try:
        with urllib.request.urlopen(url, timeout=30) as r:
            return json.load(r)["sha"][:7]
    except Exception:
        return "unknown"


def page_box(page, element):
    box = element.bounding_box()
    if not box:
        return None
    sx, sy = page.evaluate("() => [window.scrollX, window.scrollY]")
    return {**box, "x": box["x"] + sx, "y": box["y"] + sy}


def paint_everything(page, pause):
    """Scroll the whole document so GitHub paints every lazy band."""
    height = page.evaluate("() => document.body.scrollHeight")
    step = page.viewport_size["height"] // 3
    for y in range(0, height + step, step):
        page.evaluate("(y) => window.scrollTo(0, y)", y)
        page.wait_for_timeout(pause)
    page.evaluate("() => window.scrollTo(0, 0)")
    page.wait_for_timeout(600)


def unstick(page):
    """Take sticky and fixed elements out of the way of a banded capture."""
    page.evaluate("""() => {
        for (const el of document.querySelectorAll('*')) {
            const pos = getComputedStyle(el).position;
            if (pos === 'sticky') el.style.position = 'static';
            else if (pos === 'fixed') el.style.visibility = 'hidden';
        }
    }""")


def crop_box(page, selectors):
    boxes = [b for b in (page_box(page, page.query_selector(s))
                         for s in selectors if page.query_selector(s)) if b]
    if not boxes:
        return None
    # The right edge is clamped to the viewport: past it a banded capture
    # photographs nothing (a black strip down the side of every piece).
    vw = page.viewport_size["width"]
    return (max(min(b["x"] for b in boxes) - PAD, 0),
            max(min(b["y"] for b in boxes) - PAD, 0),
            min(max(b["x"] + b["width"] for b in boxes) + PAD, vw),
            max(b["y"] + b["height"] for b in boxes) + PAD)


def stitch(page, target, box):
    """Photograph the page a viewport at a time and join the bands."""
    left, top, right, bottom = box
    scale = page.evaluate("() => window.devicePixelRatio")
    band = page.viewport_size["height"]
    canvas = Image.new("RGB", (round((right - left) * scale),
                               round((bottom - top) * scale)), "white")
    y = top
    while True:
        page.evaluate("(y) => window.scrollTo(0, y)", y)
        page.wait_for_timeout(700)
        at = page.evaluate("() => window.scrollY")
        shot = Image.open(io.BytesIO(page.screenshot())).convert("RGB")
        canvas.paste(shot.crop((round(left * scale), 0,
                                round(right * scale), shot.height)),
                     (0, round((at - top) * scale)))
        if at + band >= bottom:
            break
        y = at + band
    canvas.save(target)


def hide_diagram_controls(page):
    """Hide the zoom/pan/copy buttons GitHub overlays on each diagram."""
    page.evaluate("""() => {
        const s = document.createElement('style');
        s.textContent = '.js-render-block-actions {display:none !important}';
        document.head.appendChild(s);
    }""")
    for frame in page.frames:
        if "viewscreen.githubusercontent.com" in frame.url:
            try:
                frame.evaluate("""() => {
                    const s = document.createElement('style');
                    s.textContent = 'button, [role=button], .btn {display:none !important}';
                    document.head.appendChild(s);
                }""")
            except Exception:
                pass


def shoot(page, target, banded=False):
    box = crop_box(page, FILE_VIEW)
    if box is None:
        page.screenshot(path=str(target), full_page=True)
        return False
    left, top, right, bottom = box
    scale = page.evaluate("() => window.devicePixelRatio")
    # A full-page clip does not paint the diagram iframes (they stay white),
    # so a page with diagrams is always photographed band by band.
    if banded or (bottom - top) * scale > MAX_TEXTURE:
        unstick(page)
        stitch(page, target, box)
        return True
    page.screenshot(path=str(target), full_page=True, clip={
        "x": left, "y": top, "width": right - left, "height": bottom - top})
    return True


def mermaid_state(page):
    """(diagram blocks on the page, blocks whose iframe has drawn).

    A diagram block is a section[data-type=mermaid]. Until the viewer script
    has run, it is a placeholder with no iframe; once it has, the iframe has a
    non-trivial height.
    """
    return page.evaluate("""() => {
        const blocks = [...document.querySelectorAll('[data-type="mermaid"], .js-render-enrichment-target')];
        const frames = [...document.querySelectorAll('[data-type="mermaid"] iframe, .js-render-enrichment-target iframe')];
        const drawn = frames.filter(f => f.getBoundingClientRect().height > 60).length;
        return [Math.max(blocks.length, frames.length), drawn];
    }""")


def wait_for_mermaid(page, expected):
    """Scroll each diagram into view and wait until `expected` have drawn."""
    if expected == 0:
        return 0, 0, []
    drawn = 0
    for _ in range(40):
        page.evaluate("""() => {
            for (const f of document.querySelectorAll('[data-type="mermaid"], .js-render-enrichment-target'))
                f.scrollIntoView({block: 'center'});
        }""")
        page.wait_for_timeout(700)
        total, drawn = mermaid_state(page)
        if drawn >= expected:
            break
    # Ask the frames while the diagrams are in view: out of view GitHub may
    # unload them, and then nothing is left to ask.
    errors = diagram_errors(page)
    page.evaluate("() => window.scrollTo(0, 0)")
    page.wait_for_timeout(400)
    return expected, drawn, errors


def diagram_errors(page):
    """Messages of diagrams GitHub could not draw ("Unable to render rich display").

    A parse error in the Mermaid source leaves ink on the page (the error text
    and the source), so a pixel test alone calls it drawn. Ask the diagram
    frames what they actually show.
    """
    errors = []
    # GitHub puts the error in the page itself (a flash-error box in
    # .js-render-enrichment-fallback), not in the diagram iframe.
    texts = page.evaluate("""() => [...document.querySelectorAll('.js-render-enrichment-fallback .flash-error')]
        .map(e => e.innerText)""")
    for text in texts:
        if "Unable to render" in text:
            m = re.search(r"Parse error on line (\d+)", text)
            errors.append(f"parse error on diagram line {m.group(1)}" if m else "unable to render")
    return errors


def diagram_boxes(page):
    """Page-coordinate boxes (x, y, w, h) of every Mermaid diagram block."""
    return page.evaluate("""() => [...document.querySelectorAll('section[data-type="mermaid"], section.js-render-needs-enrichment')]
        .map(e => { const r = e.getBoundingClientRect();
                    return [r.x + scrollX, r.y + scrollY, r.width, r.height]; })""")


def check_diagrams(page, target, boxes, box):
    """Crop each diagram out of the finished capture and test it is not blank.

    The iframe can report a height long before it has painted, so the only
    honest test is the pixels. Each diagram is saved as mermaid-<name>-<n>.png
    beside the capture (a check artifact, not cited by the report). Returns the
    number of diagrams whose crop holds real ink.
    """
    left, top = box[0], box[1]
    scale = page.evaluate("() => window.devicePixelRatio")
    img = Image.open(target).convert("L")
    inked = 0
    for n, (x, y, w, h) in enumerate(boxes, start=1):
        crop = img.crop((max(round((x - left) * scale), 0), max(round((y - top) * scale), 0),
                         min(round((x - left + w) * scale), img.width),
                         min(round((y - top + h) * scale), img.height)))
        if crop.width == 0 or crop.height == 0:
            continue
        dark = sum(1 for p in crop.get_flattened_data() if p < 200)
        ratio = dark / (crop.width * crop.height)
        crop.save(target.with_name(f"mermaid-{target.stem}-{n}.png"))
        if ratio > 0.004:
            inked += 1
    return inked


def render(page, name, path, ref):
    url = f"{REPO}/blob/{ref}/{path}"
    expected = source_at(ref, path).count("```mermaid")

    page.goto(url, wait_until="domcontentloaded", timeout=60_000)
    for sel in FILE_VIEW:
        page.wait_for_selector(sel, timeout=30_000)
    page.wait_for_timeout(1500)

    target = OUT / f"{name}.png"
    holes, clipped, errors = [], True, []
    for attempt, pause in enumerate(PAUSES, start=1):
        paint_everything(page, pause)
        expected, drawn, errors = wait_for_mermaid(page, expected)
        paint_everything(page, pause)
        hide_diagram_controls(page)
        clipped = shoot(page, target, banded=expected > 0)
        holes = blank_runs(target)
        if not holes:
            break
        lost = sum(n for _, n in holes)
        print(f"  {name:<14} attempt {attempt} at {pause} ms: {len(holes)} "
              f"hole(s), {lost} px unpainted - retrying slower")

    if expected:
        drawn = check_diagrams(page, target, diagram_boxes(page), crop_box(page, FILE_VIEW))
        drawn = max(drawn - len(errors), 0)
    doubled = repeats(target) if clipped else []
    width, height = Image.open(target).size
    status = []
    if not clipped:
        status.append("FULL PAGE (selectors missed)")
    if doubled:
        status.append(f"REPEATED CONTENT in {len(doubled)} block(s)")
    if holes:
        status.append(f"STILL HAS HOLES ({len(holes)})")
    for e in errors:
        status.append(f"MERMAID source does not render on GitHub: {e}")
    if expected and drawn < expected and not errors:
        status.append(f"MERMAID {drawn}/{expected} drawn")
    mm = f"mermaid {drawn}/{expected}" if expected else "no diagrams"

    meta = {"name": name, "file": path, "ref": ref, "url": url,
            "commit": commit_at(ref), "mermaid_expected": expected,
            "mermaid_drawn": drawn, "width": width, "height": height,
            "problems": status}
    (OUT / f"{name}.json").write_text(json.dumps(meta, indent=2), encoding="utf-8")
    print(f"  {name:<14} {width}x{height}  {mm}  "
          f"{'; '.join(status) if status else 'ok'}")


def main():
    args = sys.argv[1:]
    ref = "sprint-1"
    if "--ref" in args:
        i = args.index("--ref")
        ref = args[i + 1]
        del args[i:i + 2]
    global WIDTH
    if "--width" in args:
        i = args.index("--width")
        WIDTH = int(args[i + 1])
        del args[i:i + 2]
    wanted = args or list(DOCS)
    unknown = [w for w in wanted if w not in DOCS]
    if unknown:
        sys.exit(f"unknown: {', '.join(unknown)}\nknown: {', '.join(DOCS)}")

    OUT.mkdir(parents=True, exist_ok=True)
    print(f"ref {ref} -> {OUT}")
    with sync_playwright() as p:
        browser = launch(p)
        page = browser.new_page(viewport={"width": WIDTH, "height": 1000},
                                device_scale_factor=2)
        for name in wanted:
            render(page, name, DOCS[name], ref)
        browser.close()


if __name__ == "__main__":
    main()
