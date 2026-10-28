import ultraciteOxfmt from "ultracite/oxfmt";
import antiSlop from "ultracite/oxlint/anti-slop";
import ultraciteOxlintCore from "ultracite/oxlint/core";
import ultraciteOxlintReact from "ultracite/oxlint/react";
import ultraciteOxlintTanstack from "ultracite/oxlint/tanstack";
import { defineConfig } from "vite-plus";

const config = defineConfig({
  fmt: {
    ...ultraciteOxfmt,
    sortTailwindcss: {
      functions: ["clsx", "cva", "tw", "twMerge", "cn", "twJoin", "tv"],
      stylesheet: "apps/web/src/styles.css",
    },
  },
  lint: {
    extends: [
      ultraciteOxlintCore,
      ultraciteOxlintReact,
      ultraciteOxlintTanstack,
      antiSlop,
    ],
    ignorePatterns: ultraciteOxlintCore.ignorePatterns,
    jsPlugins: [{ name: "vite-plus", specifier: "vite-plus/oxlint-plugin" }],
    options: { typeAware: true, typeCheck: true },
    rules: {
      "no-unmodified-loop-condition": "error",
      "vite-plus/prefer-vite-plus-imports": "error",
    },
    overrides: [
      {
        files: ["apps/*/src/**"],
        rules: {
          "import/no-relative-parent-imports": "error",
        },
      },
    ],
  },
  staged: {
    "*": "vp check --fix",
  },
});

export default config;
