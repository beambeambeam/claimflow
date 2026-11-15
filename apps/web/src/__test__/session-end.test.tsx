import { screen, waitFor } from "@testing-library/react";
import { http, HttpResponse } from "msw";
import { setupServer } from "msw/node";
import { afterEach, describe, expect, it } from "vite-plus/test";

import {
  getMeOptions,
  loginMutation,
} from "#/lib/api/@tanstack/react-query.gen";
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
  // My Claims loads its list when /claims renders.
  http.get(`${apiUrl}/claims`, () =>
    HttpResponse.json({ data: [], page: 1, pageSize: 20, total: 0 })
  ),
  http.post(`${apiUrl}/auth/login`, () =>
    HttpResponse.json(
      { ...unauthenticated, instance: "/api/v1/auth/login" },
      { status: 401 }
    )
  ),
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
    // The first render of a route loads its code chunks, which is slow on a busy machine.
    await screen.findByRole("banner", {}, { timeout: 10_000 });

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
    await waitFor(() => {
      expect(queryClient.getQueryCache().getAll()).toHaveLength(0);
    });
  }, 20_000);

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

  it("F01-AC-21: clears the query cache and goes to login when a mutation returns 401", async () => {
    sessionUser = requester;
    const { router } = renderAt("/claims");
    await screen.findByRole("banner");

    const { queryClient } = router.options.context;
    await queryClient
      .getMutationCache()
      .build(queryClient, loginMutation())
      .execute({ body: { username: "req.anan", password: "wrong-password" } })
      .catch(() => null);

    await waitFor(() => {
      expect(router.state.location.href).toBe("/login?redirect=%2Fclaims");
    });
    await waitFor(() => {
      expect(queryClient.getQueryCache().getAll()).toHaveLength(0);
    });
  });

  it("F01-AC-21: a failed login on the login page keeps its inline error and the location", async () => {
    const { router, user } = renderAt("/login");

    await user.type(await screen.findByLabelText("Username"), "req.anan");
    await user.type(screen.getByLabelText("Password"), "wrong-password");
    await user.click(screen.getByRole("button", { name: "Log in" }));

    expect(
      await screen.findByText("Invalid username or password.")
    ).toBeTruthy();
    expect(router.state.location.href).toBe("/login");
  });

  it("F01-AC-07: a 401 on logout goes to login and clears the cache", async () => {
    sessionUser = requester;
    server.use(
      http.post(`${apiUrl}/auth/logout`, () =>
        HttpResponse.json(unauthenticated, { status: 401 })
      )
    );
    const { router, user } = renderAt("/claims");

    await user.click(await screen.findByRole("button", { name: "Log out" }));

    await waitFor(() => {
      expect(router.state.location.pathname).toBe("/login");
    });
    expect(
      router.options.context.queryClient.getQueryCache().getAll()
    ).toHaveLength(0);
  });

  it.each([
    {
      name: "403",
      response: () =>
        HttpResponse.json(
          {
            ...unauthenticated,
            status: 403,
            code: "FORBIDDEN",
            title: "Forbidden",
          },
          { status: 403 }
        ),
    },
    { name: "a network failure", response: () => HttpResponse.error() },
  ])(
    "F01-AC-07: a logout that fails with $name keeps the session, stays on the page, and shows an error",
    async ({ response }) => {
      sessionUser = requester;
      server.use(http.post(`${apiUrl}/auth/logout`, response));
      const { router, user } = renderAt("/claims");

      await user.click(await screen.findByRole("button", { name: "Log out" }));

      expect(await screen.findByText("Could not log out.")).toBeTruthy();
      expect(router.state.location.pathname).toBe("/claims");
      expect(
        router.options.context.queryClient.getQueryCache().getAll()
      ).not.toHaveLength(0);
    }
  );
});
