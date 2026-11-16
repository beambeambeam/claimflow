// Run the Go tests of apps/api (needs Docker) and keep the JSON event stream
// for gen-ac-table.ts. No shell redirection, so the file is UTF-8 everywhere.
//
//   bun run go-test 01      writes <sprint>/build/go.json
import { spawnSync } from "node:child_process";
import { mkdirSync, writeFileSync } from "node:fs";
import path from "node:path";

import { ROOT } from "./lib/paths.ts";
import { loadSprint } from "./lib/sprint.ts";

const sprint = await loadSprint(process.argv[2]);

const run = spawnSync("go", ["test", "-json", "-count=1", "./..."], {
  cwd: path.join(ROOT, "apps/api"),
  encoding: "utf-8",
  maxBuffer: 512e6,
  stdio: ["ignore", "pipe", "inherit"],
});

if (run.error) {
  throw run.error;
}

mkdirSync(sprint.build, { recursive: true });

const file = path.join(sprint.build, "go.json");

writeFileSync(file, run.stdout);

console.log(`${file} written; go test exit status ${run.status}`);
