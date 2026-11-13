import ultraciteOxlintReact from "ultracite/oxlint/react";
import ultraciteOxlintTanstack from "ultracite/oxlint/tanstack";
import type { OxlintOverride } from "vite-plus/lint";

// Web-only lint and format rules. The root vite.config.ts applies them to apps/web.
const root = "apps/web";

// Varlock generates `env.d.ts` from `.env.schema`, `bun run generate` writes the API client,
// and `astryx theme add` writes `astryx-themes.ts`.
export const webIgnorePatterns = [
  `${root}/env.d.ts`,
  `${root}/src/lib/api/**`,
  `${root}/src/astryx-themes.ts`,
];

export const webLintOverrides: OxlintOverride[] = [
  {
    files: [`${root}/**`],
    plugins: ultraciteOxlintReact.plugins,
    rules: {
      ...ultraciteOxlintReact.rules,
      ...ultraciteOxlintTanstack.rules,
    },
  },
  ...(ultraciteOxlintTanstack.overrides ?? []).map((override) => ({
    ...override,
    files: override.files.map((pattern) => `${root}/${pattern}`),
  })),
  {
    files: [`${root}/src/**`],
    rules: {
      "import/no-relative-parent-imports": "error",
    },
  },
];
