// Run a sprint's demo.ts against a running local stack and save its
// screenshots to <sprint>/figures/demo. It starts nothing: Postgres with the
// migrations and `seed` applied, the API, and the web dev server must be up.
//
//   bun run capture-demo sprint-1
//
// Env (docs/reports/.env, loaded by varlock): SEED_USER_PASSWORD, the value
// used for `seed`; BASE_URL, the web address.
import { existsSync } from "node:fs";
import path from "node:path";
import { pathToFileURL } from "node:url";

import { app } from "./lib/app.ts";
import type { Demo } from "./lib/app.ts";
import { launch } from "./lib/browser.ts";
import { loadSprint } from "./lib/sprint.ts";

const sprint = await loadSprint(process.argv[2]);
const file = path.join(sprint.dir, "demo.ts");

if (!existsSync(file)) {
  throw new Error(`no demo.ts in ${sprint.dir}`);
}

const password = process.env.SEED_USER_PASSWORD;

if (password === undefined || password === "") {
  throw new Error(
    "set SEED_USER_PASSWORD in docs/reports/.env to the value used for `seed`"
  );
}

const base = process.env.BASE_URL ?? "http://localhost:3000";
const out = path.join(sprint.figures, "demo");

const mod: unknown = await import(pathToFileURL(file).href);

const isDemoModule = (m: unknown): m is { default: Demo } =>
  typeof m === "object" &&
  m !== null &&
  "default" in m &&
  typeof m.default === "function";

if (!isDemoModule(mod)) {
  throw new Error(`${file} must export a default demo function`);
}

const browser = await launch();

try {
  const context = await browser.newContext({
    viewport: { height: 1000, width: 1280 },
  });

  await mod.default(app(await context.newPage(), base, password, out));
  console.log(`Screenshots written to ${out}`);
} finally {
  await browser.close();
}
