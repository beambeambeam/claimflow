import { createFileRoute } from "@tanstack/react-router";

import { HomePage } from "#/features/home/components/home-page";
import { getHealthOptions } from "#/lib/api/@tanstack/react-query.gen";

export const Route = createFileRoute("/")({
  // Fetches health once and reuses the cached data, like `ensureQueryData`.
  loader: async ({ context }) =>
    await context.queryClient.query({
      ...getHealthOptions(),
      staleTime: "static",
    }),
  component: HomePage,
});
