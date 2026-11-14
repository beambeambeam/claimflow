"""
Find content that appears twice in a capture.

A full-page screenshot of a very long page is assembled by the browser in
pieces. When that goes wrong the result is not a gap, which check_blank.py would
catch, but a *repeat*: the same band of the document rendered twice at different
offsets. In a sliced report that shows up as two figures carrying the same
section, which is worse than a missing one because nothing looks broken.

The tell is cheap to compute. Hash each row of pixels, then look for a long run
of consecutive rows whose hashes appear again further down.

    python check_repeat.py figures/docs
    python check_repeat.py figures/docs/tests-md.png

Reports each repeated block with both offsets and its height.
"""

import hashlib
import re
import sys
from collections import defaultdict
from pathlib import Path

from PIL import Image

Image.MAX_IMAGE_PIXELS = None

# A repeat shorter than this is ordinary: blank rows, table rules, and the flat
# background of a page legitimately recur all over a document.
MIN_RUN = 200
# Pieces written by slice_tall.py, which are crops of a capture already
# scanned here and so need no scan of their own. It has to be anchored and
# numeric: a plain "-p" substring also matches spec-pr-merged.png, which
# quietly went unscanned while the tool reported itself clean.
PIECE = re.compile(r"-p\d+$")


def row_hashes(path):
    img = Image.open(path).convert("RGB")
    width, height = img.size
    data = img.tobytes()
    stride = width * 3
    return [hashlib.blake2b(data[y * stride:(y + 1) * stride], digest_size=8).digest()
            for y in range(height)], width, height


def repeats(path):
    """(first offset, second offset, length) for each long duplicated block."""
    hashes, _width, height = row_hashes(path)

    # Rows that are one flat colour recur everywhere and mean nothing; a row
    # that appears more than a handful of times is background, not content.
    seen = defaultdict(list)
    for y, h in enumerate(hashes):
        seen[h].append(y)
    common = {h for h, ys in seen.items() if len(ys) > 8}

    found, y = [], 0
    while y < height:
        h = hashes[y]
        if h in common or len(seen[h]) < 2:
            y += 1
            continue

        # Extend the longest match against each later occurrence of this row.
        best = (0, 0)
        for other in seen[h]:
            if other <= y:
                continue
            run = 0
            while (y + run < other and other + run < height
                   and hashes[y + run] == hashes[other + run]):
                run += 1
            if run > best[0]:
                best = (run, other)

        run, other = best
        if run >= MIN_RUN:
            found.append((y, other, run))
            y += run
        else:
            y += 1
    return found


def main():
    target = Path(sys.argv[1] if len(sys.argv) > 1 else "figures/docs")
    files = ([p for p in sorted(target.glob("*.png")) if not PIECE.search(p.stem)]
             if target.is_dir() else [target])

    bad = 0
    for path in files:
        found = repeats(path)
        if not found:
            continue
        height = Image.open(path).size[1]
        total = sum(run for _, _, run in found)
        print(f"  {path.name:<24} {len(found)} repeated block(s), "
              f"{total} px ({total / height:.0%} of the capture)")
        for first, second, run in found[:8]:
            print(f"      rows {first}-{first + run} appear again at "
                  f"{second}-{second + run}  ({run} px, offset {second - first})")
        bad += 1

    print(f"{len(files)} captures, {bad} with repeated content")


if __name__ == "__main__":
    main()
