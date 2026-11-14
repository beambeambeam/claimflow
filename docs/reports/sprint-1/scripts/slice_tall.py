"""
Cut tall captures into page-sized pieces.

A rendered specification is thousands of pixels tall. Shrunk onto one A4 page
it is unreadable; left whole it overflows. This cuts each tall image into
pieces about the shape of a page, choosing a blank row for every cut so no line
of text is sliced in half.

    python scripts/slice_tall.py                  # figures/docs (GitHub captures stay whole)
    python scripts/slice_tall.py srs plan         # just those (figures/docs)

Originals stay. Pieces are written beside them as name-p1.png, name-p2.png, ...
A rerun deletes stale pieces first, so a document that got shorter leaves no
orphan piece for the report to cite.
"""

import re
import sys
from pathlib import Path

from PIL import Image

Image.MAX_IMAGE_PIXELS = None

# Anything taller than this many times its width gets cut. The number is the
# shape of the box the report draws a figure in, not a round number: 22 cm of
# height over 0.96 of a 17 cm text width. Cut shorter than that and every page
# carries a band of white; cut longer and the figure is scaled down to fit.
MAX_ASPECT = 1.30
# How far either side of the ideal cut to hunt for a blank row.
SEARCH = 0.12

# Pieces this script wrote on a previous run. Slicing an already-sliced folder
# would otherwise cut the pieces into pieces, and a capture that got shorter
# would leave the extra tail behind as a figure the report still cites.
PIECE = re.compile(r"-p\d+$")


def blank_rows(img):
    """Row indices that are a single flat colour — safe places to cut."""
    px = img.convert("L").load()
    w, h = img.size
    step = max(w // 60, 1)          # sample across the row rather than every pixel
    out = set()
    for y in range(h):
        first = px[0, y]
        if all(abs(px[x, y] - first) < 6 for x in range(0, w, step)):
            out.add(y)
    return out


def slice_image(path):
    for stale in path.parent.glob(f"{path.stem}-p*.png"):
        stale.unlink()

    img = Image.open(path)
    w, h = img.size
    if h <= w * MAX_ASPECT:
        return []

    target = int(w * MAX_ASPECT)
    blanks = blank_rows(img)

    # Cut into equal pieces rather than filling each to `target` and letting the
    # remainder be whatever is left. Taking as much as possible each time ends
    # the run on a sliver --- one document came out with a final piece 56 pixels
    # tall, a figure and a caption for one line of text. The count is the same
    # either way, so spread the height across it.
    count = -(-h // target)          # ceil: pieces no taller than target
    window = int(target * SEARCH)
    cuts = []
    for n in range(1, count):
        ideal = h * n // count
        candidates = [c for c in range(ideal - window, ideal + window) if c in blanks]
        cuts.append(min(candidates, key=lambda c: abs(c - ideal)) if candidates else ideal)

    pieces, bounds = [], [0, *cuts, h]
    for i in range(len(bounds) - 1):
        piece = path.with_name(f"{path.stem}-p{i + 1}.png")
        img.crop((0, bounds[i], w, bounds[i + 1])).save(piece)
        pieces.append(piece)
    return pieces


def main():
    args = sys.argv[1:]
    base = Path(__file__).resolve().parent.parent / "figures"
    folders = [base / "docs"]

    total = 0
    for folder in folders:
        if not folder.is_dir():
            continue
        if args:
            paths = [folder / f"{name}.png" for name in args if (folder / f"{name}.png").exists()]
        else:
            paths = sorted(p for p in folder.glob("*.png") if not PIECE.search(p.stem))
        print(folder)
        for path in paths:
            pieces = slice_image(path)
            if not pieces:
                print(f"  {path.stem:<24} fits on one page, left alone")
                continue
            total += 1
            sizes = " ".join(str(Image.open(p).size[1]) for p in pieces)
            print(f"  {path.stem:<24} -> {len(pieces)} pieces   heights: {sizes}")
    print(f"{total} captures cut")


if __name__ == "__main__":
    main()
