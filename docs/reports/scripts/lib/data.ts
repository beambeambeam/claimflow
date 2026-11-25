import { readFileSync } from "node:fs";

import { z } from "zod";

// Shapes of the JSON files the scripts hand to each other. Reading through a
// schema stops a stale or hand-edited file at the first script that reads it.

// figures/docs/<name>.json, written by render-docs next to each capture.
export const Sidecar = z.object({
  commit: z.string(),
  file: z.string(),
  mermaid_drawn: z.number(),
  mermaid_expected: z.number(),
  name: z.string().optional(),
  problems: z.array(z.string()),
  ref: z.string(),
});

// content/data/team.json, written by collect-team.
export const Team = z.object({
  issues: z.array(
    z.object({ assignees: z.array(z.string()), number: z.number() })
  ),
  prs: z.array(
    z.object({
      author: z.string(),
      number: z.number(),
      reviews: z.record(z.string(), z.array(z.string())),
      title: z.string(),
    })
  ),
});

// One line of build/go.json, the `go test -json` event stream.
export const GoEvent = z.object({
  Action: z.string(),
  Test: z.string().optional(),
});

export const parseJson = <T>(text: string, schema: z.ZodType<T>): T =>
  schema.parse(JSON.parse(text));

export const readJson = <T>(file: string, schema: z.ZodType<T>): T =>
  parseJson(readFileSync(file, "utf-8"), schema);

export type SidecarMeta = z.infer<typeof Sidecar>;
