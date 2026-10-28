import tailwindcss from "@tailwindcss/vite";
import { devtools } from "@tanstack/devtools-vite";
import { tanstackRouter } from "@tanstack/router-plugin/vite";
import viteReact from "@vitejs/plugin-react";
import { defineConfig, lazyPlugins, loadEnv } from "vite-plus";

const defaultApiUrl = "http://localhost:8080";

const config = defineConfig(({ mode }) => {
  const env = loadEnv(mode, import.meta.dirname, "");

  return {
    plugins: lazyPlugins(() => [
      devtools(),
      tailwindcss(),
      tanstackRouter({ autoCodeSplitting: true, target: "react" }),
      viteReact(),
    ]),
    resolve: { tsconfigPaths: true },
    server: { proxy: { "/api": env.API_URL || defaultApiUrl } },
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
  };
});

export default config;
