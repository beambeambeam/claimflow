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
} from "vite-plus/test";

import { client } from "#/lib/api/client.gen";
import type { CurrentUser } from "#/lib/api/types.gen";
import { zLoginRequest } from "#/lib/api/zod.gen";
import { getRouter } from "#/router";

type Role = CurrentUser["role"];

const apiUrl = "http://localhost/api/v1";
const password = "correct-password";
const roles: Role[] = ["requester", "reviewer", "finance"];

// Permissions follow the header table in F-01 §2.2.
const users: Record<Role, CurrentUser> = {
  requester: {
    id: "usr-requester",
    username: "req.anan",
    name: "Anan Requester",
    role: "requester",
    permissions: { claim: ["create", "list"] },
  },
  reviewer: {
    id: "usr-reviewer",
    username: "rev.nari",
    name: "Nari Reviewer",
    role: "reviewer",
    permissions: { claim: ["create", "list"], review: ["list"] },
  },
  finance: {
    id: "usr-finance",
    username: "fin.somchai",
    name: "Somchai Finance",
    role: "finance",
    permissions: { review: ["list"] },
  },
};

// Landing pages from F-01 §2.4.
const landingPaths: Record<Role, string> = {
  requester: "/claims",
  reviewer: "/reviews",
  finance: "/reviews",
};

const unauthenticated = {
  type: "about:blank",
  title: "Unauthorized",
  status: 401,
  code: "UNAUTHENTICATED",
  instance: "/api/v1/auth/me",
  requestId: "req-me-401",
};

const invalidCredentials = {
  type: "about:blank",
  title: "Unauthorized",
  status: 401,
  code: "UNAUTHENTICATED",
  instance: "/api/v1/auth/login",
  requestId: "req-login-401",
};

let sessionUser: CurrentUser | null = null;

const server = setupServer(
  http.get(`${apiUrl}/auth/me`, () => {
    if (sessionUser === null) {
      return HttpResponse.json(unauthenticated, { status: 401 });
    }

    return HttpResponse.json({ data: sessionUser });
  }),
  http.post(`${apiUrl}/auth/login`, async ({ request }) => {
    const body = zLoginRequest.parse(await request.json());

    const user = Object.values(users).find(
      (candidate) => candidate.username === body.username
    );

    if (user === undefined || body.password !== password) {
      return HttpResponse.json(invalidCredentials, { status: 401 });
    }

    sessionUser = user;

    return HttpResponse.json({ data: user });
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

const logIn = async (user: ReturnType<typeof userEvent.setup>, role: Role) => {
  await user.type(screen.getByLabelText("Username"), users[role].username);
  await user.type(screen.getByLabelText("Password"), password);
  await user.click(screen.getByRole("button", { name: "Log in" }));
};

describe("auth routing", () => {
  it.each(roles)(
    "F01-TEST-20: returns anonymous /claims to /claims after a %s logs in",
    async (role) => {
      const { router, user } = renderAt("/claims");

      await waitFor(() => {
        expect(router.state.location.href).toBe("/login?redirect=%2Fclaims");
      });

      await logIn(user, role);

      await waitFor(() => {
        expect(router.state.location.href).toBe("/claims");
      });
    }
  );

  it.each(roles)(
    "F01-TEST-20: sends a %s without a redirect to the role landing page",
    async (role) => {
      const { router, user } = renderAt("/login");

      await logIn(user, role);

      await waitFor(() => {
        expect(router.state.location.href).toBe(landingPaths[role]);
      });
    }
  );

  it.each([
    "//evil.example/claims",
    "/\\evil.example/claims",
    "https://evil.example/claims",
    "claims",
  ])(
    "F01-TEST-20 extra: ignores unsafe redirect %s and sends a reviewer to /reviews",
    async (redirect) => {
      const { router, user } = renderAt(
        `/login?redirect=${encodeURIComponent(redirect)}`
      );

      await logIn(user, "reviewer");

      await waitFor(() => {
        expect(router.state.location.href).toBe("/reviews");
      });
    }
  );
});
