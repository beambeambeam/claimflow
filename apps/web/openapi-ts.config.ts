import { defineConfig } from "@hey-api/openapi-ts";
import { z } from "zod";

// The contract extensions that the web reads from a generated schema with `z.globalRegistry.get`: the message of a field
// error, and the BR-04 comment limit.
const extensions = z.object({
  "x-message": z.string().optional(),
  "x-max-length": z.number().optional(),
});

// Generates the API client from the contract. Run `bun run generate`; CI fails if the output differs.
export default defineConfig({
  input: "../api/openapi.yaml",
  output: "src/lib/api",
  plugins: [
    "@hey-api/client-fetch",
    {
      name: "zod",
      metadata: ({ $, node, schema }) => {
        const { "x-message": message, "x-max-length": maxLength } =
          extensions.parse(schema);

        if (message !== undefined) {
          node.prop("message", $.literal(message));
        }

        if (maxLength !== undefined) {
          node.prop("maxLength", $.literal(maxLength));
        }
      },
    },
    { name: "@hey-api/sdk", validator: "zod" },
    "@tanstack/react-query",
  ],
});
