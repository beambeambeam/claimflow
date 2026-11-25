// One check for a whole sprint report: what is still missing, and what looks wrong.
//
//   bun run check-report 01            after bun run pdf 01
//   bun run check-report 01 --quick    skip the pixel scans
//
// Reports, in this order:
//   1. pending figures: every \evidence-family figure whose file does not exist;
//   2. \pending boxes (evidence that waits for a ticket or the release);
//   3. \todo markers, grouped by what they wait for. Write each as
//      \todo{release: ...}, \todo{1045: ...}, \todo{#30: ...} or
//      \todo{board: ...}; an unprefixed one is listed as "other";
//   4. captures with blank bands or repeated content (pixel scans of
//      figures/docs and figures/github; pieces are crops and skipped);
//   5. Mermaid diagrams that did not draw, from the sidecar files;
//   6. build/identity.tex missing (bun run identity <sprint>);
//   7. overfull boxes and missing font glyphs in main.log, with the nearest page.
//
// The final PDF needs all of 1-7 empty. Exit status is 1 while anything remains.
import { existsSync, readFileSync, readdirSync } from "node:fs";
import path from "node:path";
import { parseArgs } from "node:util";

import { Sidecar, readJson } from "./lib/data.ts";
import { blankRuns, repeats } from "./lib/image.ts";
import { groups } from "./lib/match.ts";
import { PIECE, stem } from "./lib/paths.ts";
import { loadSprint } from "./lib/sprint.ts";

const CITE = /\\evidence(?:part|half|small)?\{(?<file>[^}]+)\}/gu;
const TODO = /\\todo\{(?<text>(?:[^{}]|\{[^{}]*\})*)\}/gu;

const PENDING =
  /\\pending\{(?<what>(?:[^{}]|\{[^{}]*\})*)\}\{(?<waits>(?:[^{}]|\{[^{}]*\})*)\}/gu;

const { values, positionals } = parseArgs({
  allowPositionals: true,
  options: { quick: { default: false, type: "boolean" } },
});

const sprint = await loadSprint(positionals[0]);
const read = (file: string) => readFileSync(file, "utf-8");

const list = (dir: string, ext: string) =>
  existsSync(dir)
    ? readdirSync(dir)
        .filter((n) => n.endsWith(ext))
        .toSorted()
        .map((n) => path.join(dir, n))
    : [];

const groupOf = (text: string) =>
  groups(/^\s*(?<group>release|board|1045|capture|#\d+|ticket)\b/u.exec(text))
    .group ?? "other";

let problems = 0;

// 1-3: sources
const missing: [string, string][] = [];
const pendings: [string, string, string][] = [];
const todos = new Map<string, [string, string][]>();

const sources = [
  path.join(sprint.dir, "main.tex"),
  ...list(sprint.content, ".tex"),
  ...list(sprint.data, ".tex"),
].filter((f) => existsSync(f));

for (const file of sources) {
  const text = read(file);
  const where = path.basename(file);

  for (const [, cited = ""] of text.matchAll(CITE)) {
    if (!existsSync(path.join(sprint.dir, cited))) {
      missing.push([cited, where]);
    }
  }

  for (const [, what = "", waits = ""] of text.matchAll(PENDING)) {
    pendings.push([what, waits, where]);
  }

  for (const [, todo = ""] of text.matchAll(TODO)) {
    const group = groupOf(todo);
    todos.set(group, [...(todos.get(group) ?? []), [todo, where]]);
  }
}

console.log(`1. Pending figures: ${missing.length}`);

for (const [cited, where] of missing) {
  console.log(`   ${cited}   (${where})`);
}

console.log(`2. Pending boxes: ${pendings.length}`);

for (const [what, waits, where] of pendings) {
  console.log(`   ${where}: ${what}  -> waits for: ${waits}`);
}

const nTodo = [...todos.values()].reduce((n, v) => n + v.length, 0);

console.log(`3. TODO markers: ${nTodo}`);

for (const group of [...todos.keys()].toSorted()) {
  const items = todos.get(group) ?? [];
  console.log(`   [${group}] ${items.length}`);

  for (const [todo, where] of items) {
    console.log(`      ${where}: ${todo.slice(0, 110)}`);
  }
}

problems += missing.length + pendings.length + nTodo;

// 4: captures
if (values.quick) {
  console.log("4. Captures: skipped (--quick)");
} else {
  const scans = ["docs", "github"]
    .flatMap((sub) => list(path.join(sprint.figures, sub), ".png"))
    .filter((f) => !PIECE.test(stem(f)) && !stem(f).startsWith("mermaid-"));

  const found = await Promise.all(
    scans.map(async (f) => ({
      doubled: await repeats(f),
      file: f,
      holes: await blankRuns(f),
    }))
  );

  const bad = found.filter((r) => r.holes.length > 0 || r.doubled.length > 0);

  for (const r of bad) {
    console.log(
      `   ${path.relative(sprint.dir, r.file)}: ${r.holes.length} blank band(s), ${r.doubled.length} repeated block(s)`
    );
  }

  console.log(
    `4. Captures with blank bands or repeated content: ${bad.length} of ${scans.length}`
  );
  problems += bad.length;
}

// 5: mermaid
const metas = list(path.join(sprint.figures, "docs"), ".json").map((f) =>
  readJson(f, Sidecar)
);

const known = sprint.config.knownFaults ?? {};
const allUndrawn = metas.filter((m) => m.mermaid_drawn < m.mermaid_expected);
const undrawn = allUndrawn.filter((m) => !((m.name ?? "") in known));
const disclosed = allUndrawn.length - undrawn.length;
const expected = metas.reduce((n, m) => n + m.mermaid_expected, 0);

console.log(
  `5. Mermaid diagrams not drawn: ${undrawn.length} (of ${expected} expected in captured documents; ${disclosed} disclosed in the report as a known fault of the source)`
);

for (const m of undrawn) {
  console.log(`   ${m.file}: ${m.mermaid_drawn}/${m.mermaid_expected}`);
}

problems += undrawn.length;

// 6: identity
if (existsSync(path.join(sprint.build, "identity.tex"))) {
  console.log("6. Identity: build/identity.tex present");
} else {
  console.log(
    `6. Identity: build/identity.tex missing (bun run identity ${sprint.name})`
  );
  problems += 1;
}

// 7: log
const log = path.join(sprint.dir, "main.log");

if (existsSync(log)) {
  let page = 1;
  const over: [number, string][] = [];
  const glyphs: string[] = [];

  for (const line of read(log).split(/\r?\n/u)) {
    for (const shipped of line.matchAll(/\[(?<page>\d+)[\s\]<{]/gu)) {
      page = Number(groups(shipped).page);
    }

    if (line.startsWith("Overfull")) {
      over.push([page, line.trim().slice(0, 110)]);
    }

    if (line.startsWith("Missing character")) {
      glyphs.push(line.trim());
    }
  }

  console.log(`7. Overfull boxes in main.log: ${over.length}`);

  for (const [pg, line] of over.slice(0, 25)) {
    console.log(`   near page ${pg}: ${line}`);
  }

  const unique = [...new Set(glyphs)];
  console.log(`   Missing font glyphs: ${glyphs.length}`);

  for (const line of unique.slice(0, 10)) {
    console.log(`   ${line}`);
  }

  problems += over.length + glyphs.length;
} else {
  console.log(
    `7. Overfull boxes: main.log not found (bun run pdf ${sprint.name})`
  );
}

console.log(problems === 0 ? "\nclean" : `\n${problems} item(s) remain`);

process.exitCode = problems > 0 ? 1 : 0;
