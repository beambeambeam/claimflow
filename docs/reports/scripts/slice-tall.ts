// Cut tall captures into page-sized pieces, each cut on a blank row so no line
// of text is sliced in half.
//
//   bun run slice-tall 01             every capture in figures/docs
//   bun run slice-tall 01 srs plan    just those
//
// Originals stay. Pieces are written beside them as name-p1.png, name-p2.png.
// A rerun deletes stale pieces first, so a document that got shorter leaves no
// orphan piece for the report to cite.
import { existsSync, readdirSync, rmSync } from "node:fs";
import path from "node:path";

import { pad } from "./lib/browser.ts";
import { blankRows, crop, size } from "./lib/image.ts";
import { PIECE, stem } from "./lib/paths.ts";
import { loadSprint } from "./lib/sprint.ts";

// Anything taller than this many times its width gets cut: the shape of the box
// the report draws a figure in (22 cm over 0.96 of a 17 cm text width).
const MAX_ASPECT = 1.3;

// How far either side of the ideal cut to hunt for a blank row.
const SEARCH = 0.12;

const sliceImage = async (file: string) => {
  const dir = path.dirname(file);
  const name = stem(file);

  for (const old of readdirSync(dir)) {
    if (old.startsWith(`${name}-p`) && PIECE.test(stem(old))) {
      rmSync(path.join(dir, old));
    }
  }

  const { width: w, height: h } = await size(file);

  if (h <= w * MAX_ASPECT) {
    return [];
  }

  const target = Math.floor(w * MAX_ASPECT);
  const blanks = await blankRows(file);
  // Equal pieces, not full pieces and a remainder: filling each piece ends the
  // run on a sliver (once a final piece 56 px tall, one line of text).
  const count = Math.ceil(h / target);
  const window = Math.floor(target * SEARCH);
  const cuts = [];

  for (let n = 1; n < count; n += 1) {
    const ideal = Math.floor((h * n) / count);
    let best: number | null = null;

    for (let c = ideal - window; c < ideal + window; c += 1) {
      if (
        blanks.has(c) &&
        (best === null || Math.abs(c - ideal) < Math.abs(best - ideal))
      ) {
        best = c;
      }
    }

    cuts.push(best ?? ideal);
  }

  const bounds = [0, ...cuts, h];
  const heights = [];

  for (let i = 0; i < bounds.length - 1; i += 1) {
    const top = bounds[i] ?? 0;
    const height = (bounds[i + 1] ?? h) - top;
    await crop(
      file,
      { height, left: 0, top, width: w },
      path.join(dir, `${name}-p${i + 1}.png`)
    );
    heights.push(height);
  }

  return heights;
};

const [sprintArg, ...args] = process.argv.slice(2);
const sprint = await loadSprint(sprintArg);
const folder = path.join(sprint.figures, "docs");

const files =
  args.length > 0
    ? args
        .map((n) => path.join(folder, `${n}.png`))
        .filter((f) => existsSync(f))
    : readdirSync(folder)
        .filter((f) => f.endsWith(".png") && !PIECE.test(stem(f)))
        .toSorted()
        .map((f) => path.join(folder, f));

console.log(folder);

let cut = 0;

for (const file of files) {
  const heights = await sliceImage(file);

  if (heights.length === 0) {
    console.log(`  ${pad(stem(file), 24)} fits on one page, left alone`);
  } else {
    cut += 1;
    console.log(
      `  ${pad(stem(file), 24)} -> ${heights.length} pieces   heights: ${heights.join(" ")}`
    );
  }
}

console.log(`${cut} captures cut`);
