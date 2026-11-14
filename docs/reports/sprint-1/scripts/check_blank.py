"""
Find blank bands in a captured page.

GitHub does not paint a long markdown file all at once --- rows outside the
viewport are left unrendered --- so a full-page screenshot of a big document
comes back with holes in it. The holes are white on white and a build cannot
see them; they only show up as a page of nothing in the finished PDF.

    python check_blank.py figures/docs

Reports every run of blank rows longer than a page's worth.
"""

import re
import sys
from pathlib import Path

from PIL import Image

Image.MAX_IMAGE_PIXELS = None

# A gap this many times the image width is longer than any real spacing.
GAP = 0.35

# Pieces written by slice_tall.py, which are crops of a capture already scanned
# here and so need no scan of their own. It has to be anchored and numeric:
# a plain "-p" substring also matches spec-pr-merged.png, which quietly went
# unscanned while the tool reported itself clean.
PIECE = re.compile(r"-p\d+$")


def blank_runs(path):
    """(start, length) of every blank band longer than a page's worth.

    render_docs.py calls this to check its own captures, so it returns the runs
    alone and lets the caller ask the image for its size.
    """
    img = Image.open(path).convert("L")
    width, height = img.size
    px = img.load()
    step = max(width // 40, 1)

    limit = int(width * GAP)
    runs, start = [], None
    for y in range(height):
        first = px[0, y]
        flat = all(abs(px[x, y] - first) < 6 for x in range(0, width, step))
        if flat and start is None:
            start = y
        elif not flat and start is not None:
            if y - start > limit:
                runs.append((start, y - start))
            start = None
    if start is not None and height - start > limit:
        runs.append((start, height - start))
    return runs


def main():
    target = Path(sys.argv[1] if len(sys.argv) > 1 else "figures/docs")
    files = ([p for p in sorted(target.glob("*.png")) if not PIECE.search(p.stem)]
             if target.is_dir() else [target])

    total = 0
    for path in files:
        runs = blank_runs(path)
        if not runs:
            continue
        width, height = Image.open(path).size
        lost = sum(n for _, n in runs)
        print(f"  {path.name:<24} {width}x{height}  "
              f"{len(runs)} gap(s), {lost} px ({lost / height:.0%}) blank")
        for at, n in runs[:6]:
            print(f"      at y={at:<6} {n} px")
        total += 1

    print(f"{len(files)} captures, {total} with holes")


if __name__ == "__main__":
    main()
