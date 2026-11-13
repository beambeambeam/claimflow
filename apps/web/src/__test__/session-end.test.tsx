import { RouterProvider, createMemoryHistory } from "@tanstack/react-router";
import { cleanup, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { http, HttpResponse } from "msw";
import { setupServer } from "msw/node";
import {
  afterAll,
  afterEach,
  beforeAll,
  describe,
  expect,
  it,
  vi,
} from "vite-plus/test";

import { getMeOptions } from "#/lib/api/@tanstack/react-query.gen";
import { client } from "#/lib/api/client.gen";
import type { CurrentUser } from "#/lib/api/types.gen";
import { getRouter } from "#/router";

// Third-party dev UI. Its unmount throws under jsdom, which breaks cleanup for every later test.
vi.mock("@tanstack/react-devtools", () => ({
  TanStackDevtools: () => null,
}));

vi.mock("@tanstack/react-router-devtools", () => ({
  TanStackRouterDevtoolsPanel: () => null,
}));

const apiUrl = "http://localhost/api/v1";

const requester: CurrentUser = {
  id: "usr-requester",
  username: "req.anan",
  name: "Anan Requester",
  role: "requester",
  permissions: { claim: ["create", "list"] },
};

const unauthenticated = {
  type: "about:blank",
  title: "Unauthorized",
  status: 401,
  code: "UNAUTHENTICATED",
  instance: "/api/v1/auth/me",
  requestId: "req-me-401",
};

let sessionUser: CurrentUser | null = null;

const server = setupServer(
  http.get(`${apiUrl}/auth/me`, () => {
    if (sessionUser === null) {
      return HttpResponse.json(unauthenticated, { status: 401 });
    }

    return HttpResponse.json({ data: sessionUser });
  })
);

beforeAll(() => {
  // Node fetch needs an absolute URL; the browser resolves `/api/v1` against the page.
  client.setConfig({ baseUrl: apiUrl });
  server.listen({ onUnhandledFrame: "error" });
});

afterEach(() => {
  cleanup();
  server.resetHandlers();
  sessionUser = null;
});

afterAll(() => {
  server.close();
});

const renderAt = (initialEntry: string) => {
  const router = getRouter();
  router.update({
    ...router.options,
    history: createMemoryHistory({ initialEntries: [initialEntry] }),
  });

  render(<RouterProvider router={router} />);

  return { router, user: userEvent.setup() };
};

describe("session end", () => {
  it("F01-TEST-23: clears the query cache and goes to login when a query returns 401", async () => {
    sessionUser = requester;
    const { router } = renderAt("/claims");
    await screen.findByRole("banner");

    const { queryClient } = router.options.context;
    expect(queryClient.getQueryCache().getAll()).not.toHaveLength(0);

    // The session ends on the server; the next query gets 401.
    sessionUser = null;
    await queryClient
      .query({ ...getMeOptions(), retry: false, staleTime: 0 })
      .catch(() => null);

    await waitFor(() => {
      expect(router.state.location.href).toBe("/login?redirect=%2Fclaims");
    });
    expect(queryClient.getQueryCache().getAll()).toHaveLength(0);
  });
});
