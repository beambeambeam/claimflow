import { screen, waitFor } from "@testing-library/react";
import { http, HttpResponse } from "msw";
import { setupServer } from "msw/node";
import { afterEach, describe, expect, it } from "vite-plus/test";

import { getMeOptions } from "#/lib/api/@tanstack/react-query.gen";
import type { CurrentUser } from "#/lib/api/types.gen";

import { apiUrl, renderAt, unauthenticated, serveApi } from "./support";

const requester: CurrentUser = {
  id: "usr-requester",
  username: "req.anan",
  name: "Anan Requester",
  role: "requester",
  permissions: { claim: ["create", "list"] },
};

let sessionUser: CurrentUser | null = null;
let logoutCalls = 0;

const server = setupServer(
  http.get(`${apiUrl}/auth/me`, () => {
    if (sessionUser === null) {
      return HttpResponse.json(unauthenticated, { status: 401 });
    }

    return HttpResponse.json({ data: sessionUser });
  }),
  http.post(`${apiUrl}/auth/logout`, () => {
    logoutCalls += 1;
    sessionUser = null;

    return new HttpResponse(null, { status: 204 });
  })
);

serveApi(server);

afterEach(() => {
  sessionUser = null;
  logoutCalls = 0;
});

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

  it("F01-AC-21: stays on login without a redirect when the 401 comes on the login page", async () => {
    const { router } = renderAt("/login");
    await screen.findByLabelText("Username");

    const { queryClient } = router.options.context;
    await queryClient
      .query({ ...getMeOptions(), retry: false, staleTime: 0 })
      .catch(() => null);

    await waitFor(() => {
      expect(queryClient.getQueryCache().getAll()).toHaveLength(0);
    });
    expect(router.state.location.href).toBe("/login");
  });

  it("F01-AC-07: the Log out button ends the session, clears the cache, and opens login", async () => {
    sessionUser = requester;
    const { router, user } = renderAt("/claims");

    await user.click(await screen.findByRole("button", { name: "Log out" }));

    await waitFor(() => {
      expect(router.state.location.pathname).toBe("/login");
    });
    expect(logoutCalls).toBe(1);
    expect(
      router.options.context.queryClient.getQueryCache().getAll()
    ).toHaveLength(0);
  });
});
