// Capture every Mermaid diagram of the cited documents at full size, for print.
//
// render-docs.ts photographs a whole document, so a wide diagram shrinks with
// the page and its labels cannot be read. This opens the same GitHub file
// pages and photographs each diagram block on its own at 3x pixel density.
// The report prints these after the document they belong to.
//
//   bun run capture-diagrams sprint-1 --ref ac5bce5 --label main
//
// Output: <sprint>/figures/diagrams/<name>-<n>.png and index.json.
import { mkdirSync, writeFileSync } from "node:fs";
import path from "node:path";
import { parseArgs } from "node:util";

import type { Page } from "@playwright/test";

import { launch, pad } from "./lib/browser.ts";
import {
  MERMAID_BLOCKS,
  commitAt,
  hideDiagramControls,
  mermaidCount,
  openFile,
  waitForMermaid,
} from "./lib/github.ts";
import { crop, gray } from "./lib/image.ts";
import { loadSprint } from "./lib/sprint.ts";

// Cut the white margin GitHub leaves around a centred diagram.
const trim = async (file: string, margin = 24) => {
  const img = await gray(file);
  let [left, top, right, bottom] = [img.width, img.height, -1, -1];

  for (let y = 0; y < img.height; y += 1) {
    for (let x = 0; x < img.width; x += 1) {
      if ((img.data[y * img.width + x] ?? 255) < 243) {
        left = Math.min(left, x);
        right = Math.max(right, x);
        top = Math.min(top, y);
        bottom = Math.max(bottom, y);
      }
    }
  }

  if (right < 0) {
    return;
  }

  const l = Math.max(left - margin, 0);
  const t = Math.max(top - margin, 0);
  await crop(
    file,
    {
      height: Math.min(bottom + 1 + margin, img.height) - t,
      left: l,
      top: t,
      width: Math.min(right + 1 + margin, img.width) - l,
    },
    file
  );
};

const capture = async (
  page: Page,
  out: string,
  name: string,
  file: string,
  ref: string
) => {
  const expected = await mermaidCount(ref, file);

  if (!expected) {
    return [];
  }

  await openFile(page, ref, file);
  const { drawn, errors } = await waitForMermaid(page, expected);
  await hideDiagramControls(page);
  const blocks = page.locator(MERMAID_BLOCKS);
  const shots = [];

  for (let n = 0; n < (await blocks.count()); n += 1) {
    const block = blocks.nth(n);
    await block.scrollIntoViewIfNeeded();
    await page.waitForTimeout(1200);
    const target = path.join(out, `${name}-${n + 1}.png`);
    await block.screenshot({ path: target });
    await trim(target);
    shots.push(path.basename(target));
  }

  const status = `${shots.length}/${expected} captured, ${drawn} drawn`;
  console.log(
    `  ${pad(name, 14)} ${status}${errors.length ? `, errors: ${errors.join("; ")}` : ""}`
  );

  return shots;
};

const { values, positionals } = parseArgs({
  allowPositionals: true,
  options: {
    label: { type: "string" },
    ref: { default: "main", type: "string" },
  },
});

const [sprintArg, ...names] = positionals;
const sprint = await loadSprint(sprintArg);
const OUT = path.join(sprint.figures, "diagrams");

mkdirSync(OUT, { recursive: true });

const docs: Record<string, { file: string; diagrams: string[] }> = {};

const index = {
  commit: await commitAt(values.ref),
  docs,
  ref: values.label ?? values.ref,
};

const browser = await launch();

const page = await browser.newPage({
  deviceScaleFactor: 3,
  viewport: { height: 1100, width: 1400 },
});

for (const [name, file] of Object.entries(sprint.config.docs)) {
  if (names.length > 0 && !names.includes(name)) {
    continue;
  }

  const shots = await capture(page, OUT, name, file, values.ref);

  if (shots.length > 0) {
    index.docs[name] = { diagrams: shots, file };
  }
}

await browser.close();

writeFileSync(
  path.join(OUT, "index.json"),
  `${JSON.stringify(index, null, 2)}\n`
);
