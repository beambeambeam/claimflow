import { screen, waitFor, within } from "@testing-library/react";
import type userEvent from "@testing-library/user-event";
import { http, HttpResponse } from "msw";
import { setupServer } from "msw/node";
import { afterEach, describe, expect, it } from "vite-plus/test";

import type { CurrentUser } from "#/lib/api/types.gen";
import { zLoginRequest } from "#/lib/api/zod.gen";

import { apiUrl, renderAt, unauthenticated, serveApi } from "./support";

type Role = CurrentUser["role"];

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

serveApi(server);

afterEach(() => {
  sessionUser = null;
});

const logIn = async (user: ReturnType<typeof userEvent.setup>, role: Role) => {
  // Routes load asynchronously, so wait for the form instead of querying it synchronously.
  await user.type(
    await screen.findByLabelText("Username"),
    users[role].username
  );
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

describe("app header", () => {
  // Role labels and header links follow F-01 §2.2 and AC-20.
  const roleLabels: Record<Role, string> = {
    requester: "Requester",
    reviewer: "Reviewer",
    finance: "Finance Officer",
  };

  const expectedLinks: Record<Role, string[]> = {
    requester: ["My Claims", "New Claim"],
    reviewer: ["My Claims", "New Claim", "Review Queue"],
    finance: ["Review Queue"],
  };

  it.each(roles)(
    "F01-AC-20: shows the %s name, role label and Log out button",
    async (role) => {
      sessionUser = users[role];
      renderAt("/claims");

      const header = await screen.findByRole("banner");

      expect(within(header).getByText(users[role].name)).toBeTruthy();
      expect(within(header).getByText(roleLabels[role])).toBeTruthy();
      expect(
        within(header).getByRole("button", { name: "Log out" })
      ).toBeTruthy();
    }
  );

  it.each(roles)(
    "F01-AC-20: shows only the header links the %s role may use",
    async (role) => {
      sessionUser = users[role];
      renderAt("/claims");

      const header = await screen.findByRole("banner");

      const links = within(header)
        .getAllByRole("link")
        .map((link) => link.textContent);

      expect(links).toEqual(expectedLinks[role]);
    }
  );
});
