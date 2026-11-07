import { defineConfig } from "@hey-api/openapi-ts";

// Generates the API client from the contract. Run `bun run generate`; CI fails if the output differs.
export default defineConfig({
  input: "../api/openapi.yaml",
  output: "src/lib/api",
  plugins: [
    "@hey-api/client-fetch",
    "zod",
    { name: "@hey-api/sdk", validator: "zod" },
    "@tanstack/react-query",
  ],
});
