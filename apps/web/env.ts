import { createEnv } from "@t3-oss/env-core";
import { z } from "zod";

// Validates the variables that the Vite config reads. Vite loads them from `.env` files and the shell.
export const createConfigEnv = (
  runtimeEnv: Record<string, string | undefined>
) =>
  createEnv({
    emptyStringAsUndefined: true,
    runtimeEnv,
    server: {
      // API that the dev server proxies `/api` requests to.
      API_URL: z.url().default("http://localhost:8080"),
    },
  });
