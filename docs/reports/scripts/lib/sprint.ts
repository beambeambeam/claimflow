import { existsSync } from "node:fs";
import path from "node:path";
import { pathToFileURL } from "node:url";

import { REPORTS } from "./paths.ts";

// What differs between reports. Each report folder (sprints/NN or labs/NN)
// exports one from report.config.ts; the scripts read nothing report-specific
// from anywhere else. A lab report fills only what its scripts use.
export interface SprintConfig {
  // name -> repo path of each document Parts A-E capture, in report order.
  docs: Record<string, string>;
  // content/<part>.tex -> the documents it lists, as [name, title].
  parts: Record<string, [name: string, title: string][]>;
  // Review evidence on pull request pages. anchor null is the page header.
  reviews: { group: string; name: string; pr: number; anchor: string | null }[];
  // The project board view to capture (capture-board).
  board?: string;
  // Repo folder (or folders) of the sprint's contracts, the F-0N-*.md test
  // tables (gen-ac-table).
  contracts?: string | string[];
  // Release that accepted the sprint (a tag), and for each AC id the code that
  // implements it. With both, gen-ac-table also writes completion-table.tex.
  release?: string;
  implementation?: Record<string, string>;
  // collect-team counts only pull requests merged and issues closed on or
  // after this date (YYYY-MM-DD, UTC). Omitted: everything.
  since?: string;
  // Status text of a CI-recorded check in the AC table (gen-ac-table). Needed
  // when a planned test has no manual check text of its own.
  ciCheck?: (commit: string) => string;
  // Open issues whose bodies list planned tests not implemented yet.
  tickets?: number[];
  // Status text of manual checks in the AC table, by TEST id. Others get the
  // CI-check text.
  manualChecks?: Record<string, (commit: string) => string>;
}

export interface Sprint {
  name: string;
  dir: string;
  build: string;
  content: string;
  data: string;
  figures: string;
  config: SprintConfig;
}

const isConfigModule = (mod: unknown): mod is { default: SprintConfig } =>
  typeof mod === "object" && mod !== null && "default" in mod;

// The report folder from a sprint number under sprints/ (01), a folder under
// docs/reports (labs/06), or an absolute path (LaTeX Workshop passes the folder
// of main.tex). name is the argument that selects it again.
export const loadSprint = async (arg: string | undefined): Promise<Sprint> => {
  if (arg === undefined || arg === "") {
    throw new Error("name the report first, for example: 01 or labs/06");
  }

  const dir = /^\d{2}$/u.test(arg)
    ? path.join(REPORTS, "sprints", arg)
    : path.resolve(REPORTS, arg);

  const relative = path.relative(REPORTS, dir).split(path.sep).join("/");

  const name = relative.startsWith("sprints/")
    ? relative.slice("sprints/".length)
    : relative;

  const file = path.join(dir, "report.config.ts");

  if (!existsSync(file)) {
    throw new Error(`no report.config.ts in ${dir}`);
  }

  const mod: unknown = await import(pathToFileURL(file).href);

  if (!isConfigModule(mod)) {
    throw new Error(`${file} has no default export`);
  }

  const content = path.join(dir, "content");

  return {
    build: path.join(dir, "build"),
    config: mod.default,
    content,
    data: path.join(content, "data"),
    dir,
    figures: path.join(dir, "figures"),
    name,
  };
};
