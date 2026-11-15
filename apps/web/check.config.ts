import { readdirSync } from "node:fs";

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
  // Feature boundaries (SDS §1.4.2): shared code imports no feature or route,
  // and a feature imports only itself, `auth`, and shared code.
  {
    files: [`${root}/src/components/**`, `${root}/src/lib/**`],
    rules: {
      "no-restricted-imports": [
        "error",
        {
          patterns: [
            {
              group: ["#/features/**", "#/routes/**"],
              message: "Shared code must not import features or routes.",
            },
          ],
        },
      ],
    },
  },
  ...readdirSync(new URL("src/features", import.meta.url)).map(
    (feature): OxlintOverride => ({
      files: [`${root}/src/features/${feature}/**`],
      rules: {
        "no-restricted-imports": [
          "error",
          {
            patterns: [
              {
                group: [
                  "#/routes/**",
                  "#/features/**",
                  "!#/features/auth/**",
                  `!#/features/${feature}/**`,
                ],
                message:
                  "A feature imports only itself, auth, and shared code.",
              },
            ],
          },
        ],
      },
    })
  ),
];
