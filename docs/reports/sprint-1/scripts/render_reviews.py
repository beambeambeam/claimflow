"""
Capture the review evidence of the two Issue -> PR chains in Part F.4.

Each target is an anchor on a GitHub pull request page: the review box
(#pullrequestreview-<id>) or an inline thread (#discussion_r<id>). The browser
jumps straight to the anchor and the surrounding box is cropped. Following the
anchor also expands threads GitHub collapses once they are resolved.

    python scripts/render_reviews.py              # everything
    python scripts/render_reviews.py pr59         # one chain

Chain 1: issue #27 -> feat/route-permissions -> PR #59
         author Kiatisakk, reviewer beambeambeam (comment, fix, approval)
Chain 2: issue #25 -> feat/login-current-user -> PR #56
         author beambeambeam, reviewer Kiatisakk (changes requested, approval)

Output: figures/github/<name>.png. The repository is public: no login.
The review and comment ids come from
    gh api repos/beambeambeam/claimflow/pulls/<n>/reviews   (id)
    gh api repos/beambeambeam/claimflow/pulls/<n>/comments  (id)
"""

import sys
from pathlib import Path

from playwright.sync_api import sync_playwright

PR = "https://github.com/beambeambeam/claimflow/pull"
OUT = Path(__file__).resolve().parent.parent / "figures" / "github"
PAD = 20

# (group, filename, pull request number, anchor or None for the page header)
TARGETS = [
    ("pr59", "pr59-header",           59, None),
    ("pr59", "pr59-review-comments",  59, "pullrequestreview-5896250167"),
    ("pr59", "pr59-thread-fails-open", 59, "discussion_r4567214348"),
    ("pr59", "pr59-approved",         59, "pullrequestreview-5896430284"),
    ("pr56", "pr56-header",           56, None),
    ("pr56", "pr56-changes-requested", 56, "pullrequestreview-5890718551"),
    ("pr56", "pr56-thread-auth-in-handler", 56, "discussion_r4563077254"),
    ("pr56", "pr56-approved",         56, "pullrequestreview-5892813259"),
]

HEADER = "[class*='PageLayout-Header']"


def page_box(page, element):
    box = element.bounding_box()
    if not box:
        return None
    sx, sy = page.evaluate("() => [window.scrollX, window.scrollY]")
    return {**box, "x": box["x"] + sx, "y": box["y"] + sy}


def capture(page, name, number, anchor):
    url = f"{PR}/{number}" + (f"#{anchor}" if anchor else "")
    page.goto(url, wait_until="domcontentloaded", timeout=60_000)

    if anchor is None:
        selector, fallback = HEADER, None
    elif anchor.startswith("discussion_"):
        # Climb to the thread so the code line, the comment and the reply show.
        selector, fallback = f"review-thread-collapsible:has(#{anchor})", f"#{anchor}"
    else:
        selector, fallback = f"#{anchor}", None

    for sel in [s for s in (selector, fallback) if s]:
        try:
            page.wait_for_selector(sel, timeout=30_000)
            selector = sel
            break
        except Exception:
            continue
    else:
        print(f"  {name:<30} SKIPPED: {anchor or 'header'} not on the page")
        return

    element = page.query_selector(selector)
    element.scroll_into_view_if_needed()
    page.wait_for_timeout(1200)
    box = page_box(page, element)
    if not box or box["height"] < 5:
        print(f"  {name:<30} SKIPPED: element has no size")
        return

    OUT.mkdir(parents=True, exist_ok=True)
    target = OUT / f"{name}.png"
    page.screenshot(path=str(target), full_page=True, clip={
        "x": max(box["x"] - PAD, 0), "y": max(box["y"] - PAD, 0),
        "width": box["width"] + PAD * 2, "height": box["height"] + PAD * 2})
    print(f"  {target.name:<30} {target.stat().st_size // 1024:>5} KB")


def main():
    wanted = sys.argv[1:] or sorted({t[0] for t in TARGETS})
    items = [t for t in TARGETS if t[0] in wanted]
    if not items:
        sys.exit("nothing selected; use pr59, pr56, or no argument")
    print(f"{len(items)} captures -> {OUT}")
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
        page = browser.new_page(viewport={"width": 1100, "height": 1200},
                                device_scale_factor=2)
        for _group, name, number, anchor in items:
            capture(page, name, number, anchor)
        browser.close()


if __name__ == "__main__":
    main()
