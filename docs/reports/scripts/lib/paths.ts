import path from "node:path";

// docs/reports, the root of every sprint report, and the repository root.
export const REPORTS = path.resolve(import.meta.dirname, "../..");
export const ROOT = path.resolve(REPORTS, "../..");

// Pieces written by slice-tall.ts (name-p1.png, ...). Anchored and numeric: a
// plain "-p" substring also matches spec-pr-merged.png.
export const PIECE = /-p\d+$/u;

export const stem = (file: string) => path.basename(file, path.extname(file));
