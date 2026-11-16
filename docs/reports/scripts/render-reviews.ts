// Capture the review evidence of the Issue -> PR chains in Part F.4.
//
// Each target is an anchor on a pull request page: a review box
// (#pullrequestreview-<id>) or an inline thread (#discussion_r<id>). Following
// the anchor also expands threads GitHub collapses once resolved. The targets
// live in the sprint's report.config.ts.
//
//   bun run render-reviews sprint-1              everything
//   bun run render-reviews sprint-1 pr59         one group
//
// Output: <sprint>/figures/github/<name>.png.
import { mkdirSync, statSync } from "node:fs";
import path from "node:path";

import type { Page } from "@playwright/test";

import { launch, pad, pageBox } from "./lib/browser.ts";
import { REPO } from "./lib/github.ts";
import { loadSprint } from "./lib/sprint.ts";

const PAD = 20;
const HEADER = "[class*='PageLayout-Header']";

const capture = async (
  page: Page,
  out: string,
  name: string,
  pr: number,
  anchor: string | null
) => {
  await page.goto(`${REPO}/pull/${pr}${anchor === null ? "" : `#${anchor}`}`, {
    timeout: 60_000,
    waitUntil: "domcontentloaded",
  });

  // A discussion climbs to its thread, so the code line and replies show.
  let candidates = [HEADER];

  if (anchor?.startsWith("discussion_") === true) {
    candidates = [`review-thread-collapsible:has(#${anchor})`, `#${anchor}`];
  } else if (anchor !== null) {
    candidates = [`#${anchor}`];
  }

  let selector: string | null = null;

  for (const sel of candidates) {
    try {
      await page.waitForSelector(sel, { timeout: 30_000 });
      selector = sel;
      break;
    } catch {
      // Try the fallback selector.
    }
  }

  const element = selector === null ? null : await page.$(selector);

  if (!element) {
    console.log(
      `  ${pad(name, 30)} SKIPPED: ${anchor ?? "header"} not on the page`
    );

    return;
  }

  await element.scrollIntoViewIfNeeded();
  await page.waitForTimeout(1200);
  const box = await pageBox(page, element);

  if (!box || box.height < 5) {
    console.log(`  ${pad(name, 30)} SKIPPED: element has no size`);

    return;
  }

  const target = path.join(out, `${name}.png`);
  await page.screenshot({
    clip: {
      height: box.height + PAD * 2,
      width: box.width + PAD * 2,
      x: Math.max(box.x - PAD, 0),
      y: Math.max(box.y - PAD, 0),
    },
    fullPage: true,
    path: target,
  });
  const kb = Math.floor(statSync(target).size / 1024);
  console.log(`  ${pad(`${name}.png`, 30)} ${String(kb).padStart(5)} KB`);
};

const [sprintArg, ...groups] = process.argv.slice(2);
const sprint = await loadSprint(sprintArg);
const { reviews } = sprint.config;

const items = reviews.filter(
  (r) => groups.length === 0 || groups.includes(r.group)
);

if (items.length === 0) {
  throw new Error(
    `nothing selected; groups: ${[...new Set(reviews.map((r) => r.group))].join(", ")}`
  );
}

const out = path.join(sprint.figures, "github");

mkdirSync(out, { recursive: true });

console.log(`${items.length} captures -> ${out}`);

const browser = await launch();

const page = await browser.newPage({
  deviceScaleFactor: 2,
  viewport: { height: 1200, width: 1100 },
});

for (const { name, pr, anchor } of items) {
  await capture(page, out, name, pr, anchor);
}

await browser.close();
