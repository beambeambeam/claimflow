import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen, waitFor } from "@testing-library/react";
import type { RenderResult } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { delay, http, HttpResponse } from "msw";
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

import { LoginForm } from "#/components/login-form";
import { client } from "#/lib/api/client.gen";
import type { LoginRequest } from "#/lib/api/types.gen";
import { zLoginRequest } from "#/lib/api/zod.gen";

const loginUrl = "http://localhost/api/v1/auth/login";

const invalidCredentials = {
  type: "about:blank",
  title: "Unauthorized",
  status: 401,
  code: "UNAUTHENTICATED",
  instance: "/api/v1/auth/login",
  requestId: "req-login-401",
};

let loginBodies: LoginRequest[] = [];

const server = setupServer(
  http.post(loginUrl, async ({ request }) => {
    loginBodies.push(zLoginRequest.parse(await request.json()));

    return HttpResponse.json(invalidCredentials, { status: 401 });
  })
);

beforeAll(() => {
  // Node fetch needs an absolute URL; the browser resolves `/api/v1` against the page.
  client.setConfig({ baseUrl: "http://localhost/api/v1" });
  server.listen({ onUnhandledFrame: "error" });
});

afterEach(() => {
  cleanup();
  server.resetHandlers();
  loginBodies = [];
});

afterAll(() => {
  server.close();
});

const renderLoginForm = (): RenderResult & {
  user: ReturnType<typeof userEvent.setup>;
} => {
  const onLoggedIn = vi.fn<() => void>();

  const result = render(
    <QueryClientProvider client={new QueryClient()}>
      <LoginForm onLoggedIn={onLoggedIn} />
    </QueryClientProvider>
  );

  return { ...result, user: userEvent.setup() };
};

const getLogInButton = () =>
  screen.getByRole<HTMLButtonElement>("button", { name: "Log in" });

describe("login form", () => {
  it("F01-TEST-21: shows a message for each empty field and sends no login request", async () => {
    const { user } = renderLoginForm();

    await user.click(getLogInButton());

    await screen.findByText("Enter your username.");
    screen.getByText("Enter your password.");
    expect(loginBodies).toEqual([]);
  });

  it("F01-TEST-21: keeps the username and clears the password after a 401", async () => {
    const { user } = renderLoginForm();

    await user.type(screen.getByLabelText("Username"), "req.anan");
    await user.type(screen.getByLabelText("Password"), "wrong-password");
    await user.click(getLogInButton());

    await screen.findByText("Invalid username or password.");
    expect(screen.getByLabelText<HTMLInputElement>("Username").value).toBe(
      "req.anan"
    );
    expect(screen.getByLabelText<HTMLInputElement>("Password").value).toBe("");
    expect(loginBodies).toEqual([
      { username: "req.anan", password: "wrong-password" },
    ]);
  });

  it("F01-TEST-21: disables Log in with a spinner while the login request is in flight", async () => {
    server.use(
      http.post(loginUrl, async () => {
        await delay(300);

        return HttpResponse.json(invalidCredentials, { status: 401 });
      })
    );
    const { user } = renderLoginForm();

    await user.type(screen.getByLabelText("Username"), "req.anan");
    await user.type(screen.getByLabelText("Password"), "wrong-password");
    await user.click(getLogInButton());

    await waitFor(() => {
      expect(getLogInButton().disabled).toBe(true);
    });
    screen.getByRole("status", { name: "Logging in" });

    await screen.findByText("Invalid username or password.");
    expect(getLogInButton().disabled).toBe(false);
  });
});
