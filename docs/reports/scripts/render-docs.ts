// Capture the repository documents the report cites, as GitHub renders them.
//
// Parts A-E are the repo documents themselves. GitHub already renders them
// (tables, Mermaid, task lists), so a headless browser photographs the file box
// of each file page. GitHub paints long markdown lazily, so the page is walked
// before the shot; a page past Skia's 16384 px texture limit, or one with
// diagrams, is photographed a viewport at a time and stitched.
//
//   bun run render-docs 01 --ref ac5bce5 --label main    every document
//   bun run render-docs 01 --ref main srs plan           just those
//
// Output: <sprint>/figures/docs/<name>.png and <name>.json, a sidecar with the file, the
// ref, its commit, and the Mermaid diagrams expected against drawn.
import { mkdirSync, writeFileSync } from "node:fs";
import path from "node:path";
import { parseArgs } from "node:util";

import type { Page } from "@playwright/test";
import sharp from "sharp";

import { launch, pad, pageBox } from "./lib/browser.ts";
import {
  FILE_VIEW,
  MERMAID_BLOCKS,
  commitAt,
  hideDiagramControls,
  mermaidCount,
  openFile,
  waitForMermaid,
} from "./lib/github.ts";
import type { Mermaid } from "./lib/github.ts";
import { blankRuns, crop, gray, open, repeats, size } from "./lib/image.ts";
import type { Box } from "./lib/image.ts";
import { loadSprint } from "./lib/sprint.ts";

const PAUSES = [150, 400, 900];
const MAX_TEXTURE = 16_384;
const PAD = 24;

interface Edges {
  left: number;
  top: number;
  right: number;
  bottom: number;
}

// Make wide tables wrap inside the file box instead of scrolling sideways, and
// share a wide table's width by how much text each column holds (damped, with a
// floor) on a fixed layout. With an automatic layout a column of code names
// collapses, because code may break anywhere and so asks for almost no width.
const wrapTables = async (page: Page) => {
  await page.addStyleTag({
    content: `
      .markdown-body table {display:table !important; width:100% !important;
        max-width:100% !important; table-layout:auto !important; overflow:visible !important}
      .markdown-body td, .markdown-body th {overflow-wrap:break-word}
      .markdown-body td code, .markdown-body th code {white-space:normal !important;
        overflow-wrap:anywhere}
      .markdown-body pre {white-space:pre-wrap !important; overflow-wrap:anywhere}`,
  });
  await page.evaluate(() => {
    for (const t of document.querySelectorAll<HTMLTableElement>(
      ".markdown-body table"
    )) {
      const rows = [...t.rows];
      const n = Math.max(...rows.map((r) => r.cells.length));

      if (n < 5) {
        continue;
      }

      const len = Array.from({ length: n }, () => 0);

      for (const r of rows) {
        for (const [i, c] of [...r.cells].entries()) {
          len[i] = (len[i] ?? 0) + c.textContent.length;
        }
      }

      const weight = len.map((l) => Math.max(l / rows.length, 14) ** 0.7);
      const total = weight.reduce((a, b) => a + b, 0);
      t.style.setProperty("table-layout", "fixed", "important");

      for (const [i, c] of [...(rows[0]?.cells ?? [])].entries()) {
        c.style.width = `${((100 * (weight[i] ?? 0)) / total).toFixed(1)}%`;
      }
    }
  });
  await page.waitForTimeout(500);
};

// Scroll the whole document so GitHub paints every lazy band.
const paintEverything = async (page: Page, pause: number) => {
  const height = await page.evaluate(() => document.body.scrollHeight);
  const step = Math.floor((page.viewportSize()?.height ?? 1000) / 3);

  for (let y = 0; y < height + step; y += step) {
    await page.evaluate((to) => {
      window.scrollTo(0, to);
    }, y);
    await page.waitForTimeout(pause);
  }

  await page.evaluate(() => {
    window.scrollTo(0, 0);
  });
  await page.waitForTimeout(600);
};

// Take sticky and fixed elements out of the way of a banded capture.
const unstick = async (page: Page) => {
  await page.evaluate(() => {
    for (const el of document.querySelectorAll<HTMLElement>("*")) {
      const pos = getComputedStyle(el).position;

      if (pos === "sticky") {
        el.style.position = "static";
      } else if (pos === "fixed") {
        el.style.visibility = "hidden";
      }
    }
  });
};

const cropEdges = async (page: Page): Promise<Edges | null> => {
  const boxes = [];

  for (const sel of FILE_VIEW) {
    const el = await page.$(sel);
    const box = el ? await pageBox(page, el) : null;

    if (box) {
      boxes.push(box);
    }
  }

  if (boxes.length === 0) {
    return null;
  }

  // The right edge is clamped to the viewport: past it a banded capture
  // photographs nothing (a black strip down the side of every piece).
  const vw = page.viewportSize()?.width ?? 0;

  return {
    bottom: Math.max(...boxes.map((b) => b.y + b.height)) + PAD,
    left: Math.max(Math.min(...boxes.map((b) => b.x)) - PAD, 0),
    right: Math.min(Math.max(...boxes.map((b) => b.x + b.width)) + PAD, vw),
    top: Math.max(Math.min(...boxes.map((b) => b.y)) - PAD, 0),
  };
};

const scaleOf = async (page: Page) =>
  await page.evaluate(() => window.devicePixelRatio);

// Photograph the page a viewport at a time and join the bands.
const stitch = async (page: Page, target: string, edges: Edges) => {
  const { left, top, right, bottom } = edges;
  const scale = await scaleOf(page);
  const band = page.viewportSize()?.height ?? 1000;
  const width = Math.round((right - left) * scale);
  const height = Math.round((bottom - top) * scale);
  const layers = [];
  let y = top;

  for (;;) {
    await page.evaluate((to) => {
      window.scrollTo(0, to);
    }, y);
    await page.waitForTimeout(700);
    const at = await page.evaluate(() => window.scrollY);
    const shot = await page.screenshot();
    const { height: shotHeight } = await size(shot);
    const offset = Math.round((at - top) * scale);
    const skip = Math.max(-offset, 0);
    const rows = Math.min(shotHeight - skip, height - Math.max(offset, 0));

    if (rows > 0) {
      layers.push({
        input: await open(shot)
          .extract({
            height: rows,
            left: Math.round(left * scale),
            top: skip,
            width,
          })
          .png()
          .toBuffer(),
        left: 0,
        top: Math.max(offset, 0),
      });
    }

    if (at + band >= bottom) {
      break;
    }

    y = at + band;
  }

  await sharp({
    create: { background: "white", channels: 3, height, width },
    limitInputPixels: false,
  })
    .composite(layers)
    .png()
    .toFile(target);
};

const shoot = async (page: Page, target: string, banded: boolean) => {
  const edges = await cropEdges(page);

  if (!edges) {
    await page.screenshot({ fullPage: true, path: target });

    return null;
  }

  const scale = await scaleOf(page);

  // A full-page clip does not paint the diagram iframes (they stay white), so a
  // page with diagrams is always photographed band by band.
  if (banded || (edges.bottom - edges.top) * scale > MAX_TEXTURE) {
    await unstick(page);
    await stitch(page, target, edges);
  } else {
    await page.screenshot({
      clip: {
        height: edges.bottom - edges.top,
        width: edges.right - edges.left,
        x: edges.left,
        y: edges.top,
      },
      fullPage: true,
      path: target,
    });
  }

  return edges;
};

// Crop each diagram out of the finished capture and count those with real ink.
// The iframe can report a height long before it paints, so only the pixels are
// honest. Each crop is saved as mermaid-<name>-<n>.png, a check artifact.
const inkedDiagrams = async (page: Page, target: string, edges: Edges) => {
  const boxes = await page.evaluate(
    (sel) =>
      [...document.querySelectorAll(sel)].map((e) => {
        const r = e.getBoundingClientRect();

        return [r.x + window.scrollX, r.y + window.scrollY, r.width, r.height];
      }),
    MERMAID_BLOCKS
  );

  const scale = await scaleOf(page);
  const img = await gray(target);
  let inked = 0;

  for (const [n, [x = 0, y = 0, w = 0, h = 0]] of boxes.entries()) {
    const l = Math.max(Math.round((x - edges.left) * scale), 0);
    const t = Math.max(Math.round((y - edges.top) * scale), 0);

    const box: Box = {
      height: Math.min(Math.round((y - edges.top + h) * scale), img.height) - t,
      left: l,
      top: t,
      width: Math.min(Math.round((x - edges.left + w) * scale), img.width) - l,
    };

    if (box.width <= 0 || box.height <= 0) {
      continue;
    }

    let dark = 0;

    for (let row = box.top; row < box.top + box.height; row += 1) {
      for (let col = box.left; col < box.left + box.width; col += 1) {
        if ((img.data[row * img.width + col] ?? 255) < 200) {
          dark += 1;
        }
      }
    }

    const name = `mermaid-${path.basename(target, ".png")}-${n + 1}.png`;
    await crop(target, box, path.join(path.dirname(target), name));

    if (dark / (box.width * box.height) > 0.004) {
      inked += 1;
    }
  }

  return inked;
};

const render = async (
  page: Page,
  out: string,
  name: string,
  file: string,
  ref: string,
  label: string
) => {
  const expected = await mermaidCount(ref, file);
  const url = await openFile(page, ref, file);
  await wrapTables(page);

  const target = path.join(out, `${name}.png`);
  let holes: Awaited<ReturnType<typeof blankRuns>> = [];
  let edges: Edges | null = null;
  let mermaid: Mermaid = { drawn: 0, errors: [] };

  for (const [i, pause] of PAUSES.entries()) {
    await paintEverything(page, pause);
    mermaid = await waitForMermaid(page, expected);
    await paintEverything(page, pause);
    await hideDiagramControls(page);
    edges = await shoot(page, target, expected > 0);
    holes = await blankRuns(target);

    if (holes.length === 0) {
      break;
    }

    const lost = holes.reduce((a, h) => a + h.length, 0);
    console.log(
      `  ${pad(name, 14)} attempt ${i + 1} at ${pause} ms: ${holes.length} hole(s), ${lost} px unpainted - retrying slower`
    );
  }

  let { drawn } = mermaid;
  const finalEdges = await cropEdges(page);

  if (expected && finalEdges) {
    drawn = Math.max(
      (await inkedDiagrams(page, target, finalEdges)) - mermaid.errors.length,
      0
    );
  }

  const doubled = edges ? await repeats(target) : [];
  const { width, height } = await size(target);

  const problems = [
    ...(edges ? [] : ["FULL PAGE (selectors missed)"]),
    ...(doubled.length
      ? [`REPEATED CONTENT in ${doubled.length} block(s)`]
      : []),
    ...(holes.length ? [`STILL HAS HOLES (${holes.length})`] : []),
    ...mermaid.errors.map(
      (e) => `MERMAID source does not render on GitHub: ${e}`
    ),
    ...(expected && drawn < expected && mermaid.errors.length === 0
      ? [`MERMAID ${drawn}/${expected} drawn`]
      : []),
  ];

  const meta = {
    commit: await commitAt(ref),
    file,
    height,
    mermaid_drawn: drawn,
    mermaid_expected: expected,
    name,
    problems,
    ref: label,
    url,
    width,
  };

  writeFileSync(
    path.join(out, `${name}.json`),
    `${JSON.stringify(meta, null, 2)}\n`
  );
  const mm = expected ? `mermaid ${drawn}/${expected}` : "no diagrams";
  console.log(
    `  ${pad(name, 14)} ${width}x${height}  ${mm}  ${problems.join("; ") || "ok"}`
  );
};

const { values, positionals } = parseArgs({
  allowPositionals: true,
  options: {
    // --label main: capture from a pinned commit (--ref <sha>), cite the branch.
    label: { type: "string" },
    ref: { default: "sprint-1", type: "string" },
    // CSS px. Below GitHub's 1012 px breakpoint the sidebar goes and the file
    // box fills the width, so text prints near 8 pt.
    width: { default: "1000", type: "string" },
  },
});

const [sprintArg, ...names] = positionals;
const sprint = await loadSprint(sprintArg);
const { docs } = sprint.config;
const OUT = path.join(sprint.figures, "docs");
const wanted = names.length > 0 ? names : Object.keys(docs);
const unknown = wanted.filter((w) => !(w in docs));

if (unknown.length > 0) {
  throw new Error(
    `unknown: ${unknown.join(", ")}; known: ${Object.keys(docs).join(", ")}`
  );
}

mkdirSync(OUT, { recursive: true });

console.log(`ref ${values.ref} -> ${OUT}`);

const browser = await launch();

const page = await browser.newPage({
  deviceScaleFactor: 2,
  viewport: { height: 1000, width: Number(values.width) },
});

for (const name of wanted) {
  await render(
    page,
    OUT,
    name,
    docs[name] ?? "",
    values.ref,
    values.label ?? values.ref
  );
}

await browser.close();
