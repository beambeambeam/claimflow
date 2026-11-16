// Build <sprint>/main.pdf: write build/identity.tex, then XeLaTeX twice (the
// second pass fills page numbers and the contents).
//
//   bun run pdf sprint-1
//
// Needs xelatex on PATH: MiKTeX or TeX Live on Windows, MacTeX on macOS.
import { spawnSync } from "node:child_process";
import { readFileSync } from "node:fs";
import path from "node:path";

import { loadSprint } from "./lib/sprint.ts";

const sprint = await loadSprint(process.argv[2]);

const run = (cmd: string, args: string[]) =>
  spawnSync(cmd, args, { cwd: sprint.dir, encoding: "utf-8", stdio: "pipe" });

const identity = spawnSync(
  process.execPath,
  [path.join(import.meta.dirname, "identity.ts"), sprint.dir],
  { stdio: "inherit" }
);

if (identity.status !== 0) {
  process.exit(identity.status ?? 1);
}

for (const pass of [1, 2]) {
  const xe = run("xelatex", [
    "-interaction=nonstopmode",
    "-file-line-error",
    "main.tex",
  ]);

  if (xe.error) {
    throw new Error(`xelatex not found on PATH (${xe.error.message})`);
  }

  if (xe.status !== 0) {
    const log = readFileSync(path.join(sprint.dir, "main.log"), "utf-8").split(
      /\r?\n/u
    );

    console.error(log.slice(-40).join("\n"));
    console.error(
      `xelatex pass ${pass} failed; full log: ${path.join(sprint.dir, "main.log")}`
    );
    process.exit(1);
  }
}

console.log(
  `${path.join(sprint.dir, "main.pdf")} built; check it with bun run check-report ${sprint.name}`
);
