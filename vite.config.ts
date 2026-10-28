import ultraciteOxfmt from "ultracite/oxfmt";
import antiSlop from "ultracite/oxlint/anti-slop";
import ultraciteOxlintCore from "ultracite/oxlint/core";
import { defineConfig } from "vite-plus";

import { webFmtOverride, webLintOverrides } from "./apps/web/check.config";

// Repository-wide format, lint, and pre-commit rules. App-specific rules live in each app.
const config = defineConfig({
  fmt: {
    ...ultraciteOxfmt,
    overrides: [webFmtOverride],
  },
  lint: {
    extends: [ultraciteOxlintCore, antiSlop],
    ignorePatterns: ultraciteOxlintCore.ignorePatterns,
    jsPlugins: [{ name: "vite-plus", specifier: "vite-plus/oxlint-plugin" }],
    options: { typeAware: true, typeCheck: true },
    overrides: webLintOverrides,
    rules: {
      "no-unmodified-loop-condition": "error",
      "vite-plus/prefer-vite-plus-imports": "error",
    },
  },
  staged: {
    "*": "vp check --fix",
  },
});

export default config;
