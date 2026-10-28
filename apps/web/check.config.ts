import ultraciteOxlintReact from "ultracite/oxlint/react";
import ultraciteOxlintTanstack from "ultracite/oxlint/tanstack";
import type { OxfmtOverrideConfig } from "vite-plus/fmt";
import type { OxlintOverride } from "vite-plus/lint";

// Web-only lint and format rules. The root vite.config.ts applies them to apps/web.
const root = "apps/web";

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

export const webFmtOverride: OxfmtOverrideConfig = {
  files: [`${root}/**`],
  options: {
    sortTailwindcss: {
      functions: ["clsx", "cva", "tw", "twMerge", "cn", "twJoin", "tv"],
      stylesheet: `${root}/src/styles.css`,
    },
  },
};
