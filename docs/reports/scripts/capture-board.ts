// Capture the public project board for Part F.4.
//
//   bun run capture-board 01           the board in report.config.ts
//   bun run capture-board 01 <url>     another view of the board
//
// GitHub Projects paints cards lazily and scrolls inside the board, so the
// viewport is made tall enough for the tallest column, then the empty space
// below the cards is cut. Output: <sprint>/figures/github/board.png.
import { mkdirSync, statSync } from "node:fs";
import path from "node:path";

import { launch } from "./lib/browser.ts";
import { crop, size } from "./lib/image.ts";
import { loadSprint } from "./lib/sprint.ts";

const [sprintArg, url] = process.argv.slice(2);
const sprint = await loadSprint(sprintArg);
const target = path.join(sprint.figures, "github", "board.png");

mkdirSync(path.dirname(target), { recursive: true });

const browser = await launch();

const page = await browser.newPage({
  deviceScaleFactor: 1.5,
  viewport: { height: 3400, width: 2400 },
});

const view = url ?? sprint.config.board;

if (view === undefined) {
  throw new Error(`no board in ${sprint.name}/report.config.ts; pass the URL`);
}

await page.goto(view, { timeout: 60_000, waitUntil: "domcontentloaded" });

await page.waitForTimeout(8000);

const shot = await page.screenshot();

await browser.close();

const { width, height } = await size(shot);

await crop(
  shot,
  {
    height: Math.round(height * 0.65),
    left: 0,
    top: 0,
    width: Math.round(width * 0.77),
  },
  target
);

console.log(
  `board.png ${Math.floor(statSync(target).size / 1024)} KB  ${view}`
);
