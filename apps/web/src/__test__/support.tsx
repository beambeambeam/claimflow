import { RouterProvider, createMemoryHistory } from "@tanstack/react-router";
import { cleanup, render } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import type { SetupServer } from "msw/node";
import { afterAll, afterEach, beforeAll } from "vite-plus/test";

import { client } from "#/lib/api/client.gen";
import { getRouter } from "#/router";

export const apiUrl = "http://localhost/api/v1";

// The body of a 401 from `GET /auth/me`.
export const unauthenticated = {
  type: "about:blank",
  title: "Unauthorized",
  status: 401,
  code: "UNAUTHENTICATED",
  instance: "/api/v1/auth/me",
  requestId: "req-me-401",
};

// Runs `server` for the tests of the calling file and resets it and the DOM after each test.
export const serveApi = (server: SetupServer) => {
  beforeAll(() => {
    // Node fetch needs an absolute URL; the browser resolves `/api/v1` against the page.
    client.setConfig({ baseUrl: apiUrl });
    server.listen({ onUnhandledFrame: "error" });
  });

  afterEach(() => {
    cleanup();
    server.resetHandlers();
  });

  afterAll(() => {
    server.close();
  });
};

export const renderAt = (initialEntry: string) => {
  const router = getRouter();
  router.update({
    ...router.options,
    history: createMemoryHistory({ initialEntries: [initialEntry] }),
  });

  render(<RouterProvider router={router} />);

  return { router, user: userEvent.setup() };
};
