import { RouterProvider, createMemoryHistory } from "@tanstack/react-router";
import { http, HttpResponse } from "msw";
import { setupServer } from "msw/node";
import { renderToString } from "react-dom/server";
import { afterAll, beforeAll, expect, it } from "vite-plus/test";

import { getHealthQueryKey } from "#/lib/api/@tanstack/react-query.gen";
import { client } from "#/lib/api/client.gen";
import { getRouter } from "#/router";

const health = { status: "ok", version: "v1.2.3" };

const server = setupServer(
  http.get("http://localhost/api/v1/health", () => HttpResponse.json(health))
);

beforeAll(() => {
  // Node fetch needs an absolute URL; the browser resolves `/api/v1` against the page.
  client.setConfig({ baseUrl: "http://localhost/api/v1" });
  server.listen({ onUnhandledFrame: "error" });
});

afterAll(() => {
  server.close();
});

it("uses the app's router defaults and registers the home route", () => {
  const router = getRouter();

  expect(router.options.defaultPreload).toBe("intent");
  expect(router.options.defaultPreloadStaleTime).toBe(0);
  expect(router.options.scrollRestoration).toBe(true);
  expect(router.routesByPath["/"]).toBeDefined();
});

it("loads the home page health through the router's query client", async () => {
  const router = getRouter();
  router.update({
    ...router.options,
    history: createMemoryHistory({ initialEntries: ["/"] }),
  });

  await router.load();

  expect(
    router.options.context.queryClient.getQueryData(getHealthQueryKey())
  ).toEqual(health);
  expect(renderToString(<RouterProvider router={router} />)).toContain(
    "API version v1.2.3"
  );
});
