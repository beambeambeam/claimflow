import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { renderToStaticMarkup } from "react-dom/server";
import { describe, expect, it } from "vite-plus/test";

import { HomePage } from "#/features/home/components/home-page";
import { getHealthQueryKey } from "#/lib/api/@tanstack/react-query.gen";

describe("home page", () => {
  it("shows the API version from the health query", () => {
    const queryClient = new QueryClient();
    queryClient.setQueryData(getHealthQueryKey(), {
      status: "ok",
      version: "v1.2.3",
    });

    const markup = renderToStaticMarkup(
      <QueryClientProvider client={queryClient}>
        <HomePage />
      </QueryClientProvider>
    );

    expect(markup).toContain("ClaimFlow");
    expect(markup).toContain("API version v1.2.3");
  });
});
