import type { OxlintOverride } from "vite-plus/lint";

// Report-only lint rules. The root vite.config.ts applies them to docs/reports.
const root = "docs/reports";

export const reportsLintOverrides: OxlintOverride[] = [
  {
    files: [`${root}/**`],
    rules: {
      // The scripts drive one browser page and one file at a time, in order:
      // a capture waits for the step before it.
      "no-await-in-loop": "off",
    },
  },
];
