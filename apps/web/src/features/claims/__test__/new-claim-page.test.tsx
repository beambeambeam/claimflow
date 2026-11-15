import { screen } from "@testing-library/react";
import { http, HttpResponse } from "msw";
import { setupServer } from "msw/node";
import { describe, expect, it } from "vite-plus/test";

import { apiUrl, renderAt, serveApi } from "#/__test__/support";
import type { CurrentUser } from "#/lib/api/types.gen";

const requester: CurrentUser = {
  id: "usr-requester",
  username: "req.anan",
  name: "Anan K.",
  role: "requester",
  permissions: { claim: ["create", "list"] },
};

const finance: CurrentUser = {
  id: "usr-finance",
  username: "fin.wanida",
  name: "Wanida",
  role: "finance",
  permissions: { review: ["list"], finance: ["list"], report: ["read"] },
};

// The first render of a route loads its code chunks, which is slow on a busy machine.
const slow = { timeout: 10_000 };

let sessionUser: CurrentUser = finance;

const server = setupServer(
  http.get(`${apiUrl}/auth/me`, () => HttpResponse.json({ data: sessionUser }))
);

serveApi(server);

describe("New Claim page", () => {
  it("F02-TEST-20: shows 'You can't create claims.' and no form to a Finance Officer", async () => {
    sessionUser = finance;
    renderAt("/claims/new");

    await screen.findByText("You can't create claims.", {}, slow);
    expect(screen.queryByLabelText("Title")).toBeNull();
    expect(screen.queryByRole("button", { name: "Submit" })).toBeNull();
  }, 20_000);

  it("shows the claim form to a requester", async () => {
    sessionUser = requester;
    renderAt("/claims/new");

    await screen.findByLabelText("Title", {}, slow);
    expect(screen.queryByText("You can't create claims.")).toBeNull();
    screen.getByRole("button", { name: "Submit" });
  }, 20_000);
});
