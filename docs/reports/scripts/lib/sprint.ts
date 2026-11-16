import { existsSync } from "node:fs";
import path from "node:path";
import { pathToFileURL } from "node:url";

import { REPORTS } from "./paths.ts";

// What differs between sprint reports. Each sprint folder exports one from
// report.config.ts; the scripts read nothing sprint-specific from anywhere else.
export interface SprintConfig {
  // name -> repo path of each document Parts A-E capture, in report order.
  docs: Record<string, string>;
  // content/<part>.tex -> the documents it lists, as [name, title].
  parts: Record<string, [name: string, title: string][]>;
  // Review evidence on pull request pages. anchor null is the page header.
  reviews: { group: string; name: string; pr: number; anchor: string | null }[];
  // The project board view to capture.
  board: string;
  // Repo folder of the sprint's contracts (the F-0N-*.md test tables).
  contracts: string;
  // Open issues whose bodies list planned tests not implemented yet.
  tickets: number[];
  // Status text of manual checks in the AC table, by TEST id. Others get the
  // CI-check text.
  manualChecks: Record<string, (commit: string) => string>;
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

// The sprint folder from a name (sprint-1) or a path (LaTeX Workshop passes
// the folder of main.tex).
export const loadSprint = async (arg: string | undefined): Promise<Sprint> => {
  if (arg === undefined || arg === "") {
    throw new Error("name the sprint first, for example: sprint-1");
  }

  const dir = path.resolve(REPORTS, arg);
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
    name: path.basename(dir),
  };
};
