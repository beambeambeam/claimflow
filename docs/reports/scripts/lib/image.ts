import { createHash } from "node:crypto";
import { writeFile } from "node:fs/promises";

import sharp from "sharp";
import type { Sharp } from "sharp";

// Pixel work on the captures. sharp replaces Pillow; captures run past
// libvips' default pixel limit, so every image opens without it.
export const open = (input: string | Buffer) =>
  sharp(input, { limitInputPixels: false });

export interface Raster {
  data: Buffer;
  width: number;
  height: number;
  channels: number;
}

const raw = async (image: Sharp): Promise<Raster> => {
  const { data, info } = await image
    .raw()
    .toBuffer({ resolveWithObject: true });

  return {
    channels: info.channels,
    data,
    height: info.height,
    width: info.width,
  };
};

export const gray = async (input: string | Buffer) =>
  await raw(open(input).grayscale());

export const rgb = async (input: string | Buffer) =>
  await raw(open(input).removeAlpha());

export const size = async (input: string | Buffer) => {
  const { width, height } = await open(input).metadata();

  return { height, width };
};

export interface Box {
  left: number;
  top: number;
  width: number;
  height: number;
}

// Crop to a PNG. Reads fully before writing, so target may equal input.
export const crop = async (
  input: string | Buffer,
  box: Box,
  target: string
) => {
  const png = await open(input).extract(box).png().toBuffer();
  await writeFile(target, png);
};

// A row is flat when every sampled pixel is within 6 grey levels of its first.
const flatRow = (img: Raster, y: number, step: number) => {
  const row = y * img.width;
  const first = img.data[row] ?? 0;

  for (let x = 0; x < img.width; x += step) {
    if (Math.abs((img.data[row + x] ?? 0) - first) >= 6) {
      return false;
    }
  }

  return true;
};

// Row indices that are a single flat colour: safe places to cut.
export const blankRows = async (file: string) => {
  const img = await gray(file);
  const step = Math.max(Math.floor(img.width / 60), 1);
  const out = new Set<number>();

  for (let y = 0; y < img.height; y += 1) {
    if (flatRow(img, y, step)) {
      out.add(y);
    }
  }

  return out;
};

// A gap this many times the image width is longer than any real spacing.
const GAP = 0.35;

export interface Run {
  start: number;
  length: number;
}

// Blank bands longer than a page's worth. GitHub paints a long markdown file
// lazily, so a capture can come back with white-on-white holes no build sees.
export const blankRuns = async (file: string) => {
  const img = await gray(file);
  const step = Math.max(Math.floor(img.width / 40), 1);
  const limit = Math.floor(img.width * GAP);
  const runs: Run[] = [];
  let start: number | null = null;

  for (let y = 0; y < img.height; y += 1) {
    const flat = flatRow(img, y, step);

    if (flat && start === null) {
      start = y;
    } else if (!flat && start !== null) {
      if (y - start > limit) {
        runs.push({ length: y - start, start });
      }

      start = null;
    }
  }

  if (start !== null && img.height - start > limit) {
    runs.push({ length: img.height - start, start });
  }

  return runs;
};

// A repeat shorter than this is ordinary: blank rows and table rules recur.
const MIN_RUN = 200;

export interface Repeat {
  first: number;
  second: number;
  length: number;
}

const longestMatch = (hashes: string[], y: number, others: number[]) => {
  let best = { other: 0, run: 0 };

  for (const other of others) {
    if (other <= y) {
      continue;
    }

    let run = 0;

    while (
      y + run < other &&
      other + run < hashes.length &&
      hashes[y + run] === hashes[other + run]
    ) {
      run += 1;
    }

    if (run > best.run) {
      best = { other, run };
    }
  }

  return best;
};

// Content that appears twice: the same band rendered at two offsets, which a
// broken full-page screenshot produces instead of a gap. Hash each pixel row,
// then look for long runs of rows whose hashes recur further down.
export const repeats = async (file: string) => {
  const img = await rgb(file);
  const stride = img.width * img.channels;

  const hashes = Array.from({ length: img.height }, (_, y) =>
    createHash("sha1")
      .update(img.data.subarray(y * stride, (y + 1) * stride))
      .digest("hex")
  );

  const seen = new Map<string, number[]>();

  for (const [y, h] of hashes.entries()) {
    const ys = seen.get(h);

    if (ys) {
      ys.push(y);
    } else {
      seen.set(h, [y]);
    }
  }

  const found: Repeat[] = [];
  let y = 0;

  while (y < img.height) {
    const ys = seen.get(hashes[y] ?? "") ?? [];

    // A row seen more than a handful of times is background, not content.
    if (ys.length < 2 || ys.length > 8) {
      y += 1;
      continue;
    }

    const { run, other } = longestMatch(hashes, y, ys);

    if (run >= MIN_RUN) {
      found.push({ first: y, length: run, second: other });
      y += run;
    } else {
      y += 1;
    }
  }

  return found;
};
