"""
Capture the public project board (project 5) for Part F.4.

    python scripts/capture_board.py              # figures/github/board.png
    python scripts/capture_board.py <url>        # another view of the board

The board is public, so no login. GitHub Projects paints cards lazily and
scrolls inside the board, not the page, so the viewport is made tall enough
for the tallest column. Cut with slice_tall.py if the result is taller than a page.
"""

import sys
from pathlib import Path

from playwright.sync_api import sync_playwright

URL = "https://github.com/users/beambeambeam/projects/5"
OUT = Path(__file__).resolve().parent.parent / "figures" / "github" / "board.png"


def main():
    url = sys.argv[1] if len(sys.argv) > 1 else URL
    OUT.parent.mkdir(parents=True, exist_ok=True)
    with sync_playwright() as p:
        browser = None
        for channel in ("chrome", "msedge"):
            try:
                browser = p.chromium.launch(channel=channel, headless=True)
                break
            except Exception:
                continue
        if browser is None:
            sys.exit("neither Chrome nor Edge could be launched")
        page = browser.new_page(viewport={"width": 2400, "height": 3400},
                                device_scale_factor=1.5)
        page.goto(url, wait_until="domcontentloaded", timeout=60_000)
        page.wait_for_timeout(8000)
        page.screenshot(path=str(OUT))
        # The tall viewport leaves empty column space below the cards: cut it.
        from PIL import Image
        im = Image.open(OUT)
        im.crop((0, 0, round(im.width * 0.77), round(im.height * 0.65))).save(OUT)
        print(f"{OUT.name} {OUT.stat().st_size // 1024} KB  {url}")
        browser.close()


if __name__ == "__main__":
    main()
