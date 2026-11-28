import { screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { http, HttpResponse } from "msw";
import { setupServer } from "msw/node";
import { describe, expect, it } from "vite-plus/test";

import { apiUrl, renderAt, serveApi } from "#/__test__/support";
import type { CurrentUser } from "#/lib/api/types.gen";

import { claim } from "./fixtures";

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
    screen.getByText(
      "Your account has no assigned reviewer, so it cannot submit claims."
    );
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

  // The confirmation lives in the page's state, and the header link points at the URL already open (F09-TEST-13).
  it("opens an empty form from the New Claim header link after a submit", async () => {
    sessionUser = requester;
    server.use(
      http.post(`${apiUrl}/claims`, () =>
        HttpResponse.json(
          { data: claim({ claimNo: "CLM-2026-000123" }) },
          { status: 201, headers: { ETag: '"1"' } }
        )
      )
    );
    const user = userEvent.setup();
    renderAt("/claims/new");

    await user.type(
      await screen.findByLabelText("Title", {}, slow),
      "Bangkok workshop travel"
    );
    await user.type(screen.getByLabelText("Purpose"), "Train fares and hotel");
    await user.click(screen.getAllByLabelText("Category")[0]);
    await user.click(await screen.findByRole("option", { name: "Travel" }));
    await user.type(screen.getAllByLabelText("Description")[0], "Train");
    await user.type(screen.getAllByLabelText("Expense date")[0], "2026-11-03");
    await user.type(screen.getAllByLabelText("Amount")[0], "250.00");
    await user.type(screen.getAllByLabelText("Receipt reference")[0], "SRT-1");
    await user.click(screen.getByRole("button", { name: "Submit" }));
    await screen.findByText("Claim CLM-2026-000123 submitted", {}, slow);

    await user.click(
      within(
        screen.getByRole("navigation", { name: "Main navigation" })
      ).getByRole("link", { name: "New Claim" })
    );

    const title = await screen.findByLabelText<HTMLInputElement>(
      "Title",
      {},
      slow
    );

    expect(title.value).toBe("");
  }, 30_000);
});
