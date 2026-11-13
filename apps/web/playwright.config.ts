import { defineConfig, devices } from "@playwright/test";

// E2E runs against a local stack: seeded PostgreSQL, the API on PORT (8080), and the web dev server on 3000.
// Start the stack first (see apps/api/README.md and `bun run dev`). The config does not start servers.
export default defineConfig({
  testDir: "./e2e",
  fullyParallel: false,
  workers: 1,
  reporter: "list",
  use: {
    baseURL: process.env.E2E_BASE_URL ?? "http://localhost:3000",
    trace: "retain-on-failure",
  },
  projects: [{ name: "chromium", use: { ...devices["Desktop Chrome"] } }],
});
