import tailwindcss from "@tailwindcss/vite";
import { devtools } from "@tanstack/devtools-vite";
import { tanstackRouter } from "@tanstack/router-plugin/vite";
import viteReact from "@vitejs/plugin-react";
import ultraciteOxfmt from "ultracite/oxfmt";
import antiSlop from "ultracite/oxlint/anti-slop";
import ultraciteOxlintCore from "ultracite/oxlint/core";
import ultraciteOxlintReact from "ultracite/oxlint/react";
import ultraciteOxlintTanstack from "ultracite/oxlint/tanstack";
import { defineConfig, lazyPlugins } from "vite-plus";

const config = defineConfig({
  fmt: {
    ...ultraciteOxfmt,
    sortTailwindcss: {
      functions: ["clsx", "cva", "tw", "twMerge", "cn", "twJoin", "tv"],
      stylesheet: "src/styles.css",
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
        files: ["src/**"],
        rules: {
          "import/no-relative-parent-imports": "error",
        },
      },
    ],
  },
  plugins: lazyPlugins(() => [
    devtools(),
    tailwindcss(),
    tanstackRouter({ autoCodeSplitting: true, target: "react" }),
    viteReact(),
  ]),
  resolve: { tsconfigPaths: true },
  test: {
    coverage: {
      include: ["src/**/*.{ts,tsx}"],
      exclude: [
        "src/**/__test__/**",
        "src/main.tsx",
        "src/routeTree.gen.ts",
        "src/routes/**",
      ],
      thresholds: {
        branches: 100,
        functions: 100,
        lines: 100,
        statements: 100,
      },
    },
    include: ["src/**/__test__/**/*.test.{ts,tsx}"],
  },
  staged: {
    "*": "vp check --fix",
  },
});

export default config;
