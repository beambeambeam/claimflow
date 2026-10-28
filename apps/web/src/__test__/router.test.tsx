import { expect, it } from "vite-plus/test";

import { getRouter } from "#/router";

it("uses the app's router defaults and registers the home route", () => {
  const router = getRouter();

  expect(router.options.defaultPreload).toBe("intent");
  expect(router.options.defaultPreloadStaleTime).toBe(0);
  expect(router.options.scrollRestoration).toBe(true);
  expect(router.routesByPath["/"]).toBeDefined();
});
