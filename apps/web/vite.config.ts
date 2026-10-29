import tailwindcss from "@tailwindcss/vite";
import { devtools } from "@tanstack/devtools-vite";
import { tanstackRouter } from "@tanstack/router-plugin/vite";
import { varlockVitePlugin } from "@varlock/vite-integration";
import viteReact from "@vitejs/plugin-react";
import { ENV } from "varlock/env";
import { defineConfig, lazyPlugins } from "vite-plus";

const config = defineConfig({
  plugins: [
    varlockVitePlugin(),
    lazyPlugins(() => [
      devtools(),
      tailwindcss(),
      tanstackRouter({ autoCodeSplitting: true, target: "react" }),
      viteReact(),
    ]),
  ],
  resolve: { tsconfigPaths: true },
  server: { proxy: { "/api": ENV.API_URL } },
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
});

export default config;
