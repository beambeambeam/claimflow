import { screen, within } from "@testing-library/react";
import { http, HttpResponse } from "msw";
import { setupServer } from "msw/node";
import { beforeEach, describe, expect, it } from "vite-plus/test";

import { apiUrl, renderAt, serveApi } from "#/__test__/support";
import type { Claim } from "#/lib/api/types.gen";

import { pendingClaim, reviewer } from "./fixtures";

// The first render of a route loads its code chunks, which is slow on a busy machine.
const slow = { timeout: 10_000 };

const claimId = "0192f1c4-7b2e-7c3a-9d41-3f0a6b8e2c11";

let current: Claim = pendingClaim();

const server = setupServer(
  http.get(`${apiUrl}/auth/me`, () => HttpResponse.json({ data: reviewer })),
  http.get(`${apiUrl}/claims/${claimId}`, () =>
    HttpResponse.json({ data: current })
  ),
  http.get(`${apiUrl}/claims`, () =>
    HttpResponse.json({ data: [], page: 1, pageSize: 20, total: 0 })
  ),
  http.post(`${apiUrl}/claims/${claimId}/approve`, () => {
    // The API picks the target status; the web reads it from the response.
    current = pendingClaim({ allowedActions: [] });

    return HttpResponse.json({ data: current });
  })
);

serveApi(server);

// The claim before the approval: SUBMITTED and assigned to the reviewer.
const submitted = () =>
  pendingClaim({
    status: "SUBMITTED",
    version: 0,
    history: pendingClaim().history.slice(0, 1),
    allowedActions: ["approve", "reject"],
  });

beforeEach(() => {
  current = pendingClaim();
});

describe("Pending finance", () => {
  it("F08-TEST-15: an approval that goes to Finance says so in the toast", async () => {
    current = submitted();
    const { user } = renderAt(`/claims/${claimId}`);
    await screen.findByRole("heading", { name: "CLM-2026-000123" }, slow);

    await user.click(screen.getByRole("button", { name: "Approve" }));
    await user.click(
      within(screen.getByRole("dialog")).getByRole("button", {
        name: "Approve",
      })
    );

    await screen.findAllByText(
      "Claim approved. It now waits for a final decision by Finance."
    );
  }, 30_000);

  it("F08-TEST-15: Claim Detail shows the purple Pending finance badge and the waiting notice above the history", async () => {
    current = pendingClaim({ allowedActions: [] });
    renderAt(`/claims/${claimId}`);
    await screen.findByRole("heading", { name: "CLM-2026-000123" }, slow);

    const badge = screen
      .getByText("Pending finance")
      .closest<HTMLElement>("[data-variant]");

    expect(badge?.dataset.variant).toBe("purple");

    const history = screen.getByRole("region", { name: "History" });
    within(history).getByText("Waiting for a final decision by Finance.");
  }, 20_000);

  it("F08-TEST-15: a claim that is not pending finance shows no waiting notice", async () => {
    current = submitted();
    renderAt(`/claims/${claimId}`);
    await screen.findByRole("heading", { name: "CLM-2026-000123" }, slow);

    expect(
      screen.queryByText("Waiting for a final decision by Finance.")
    ).toBeNull();
  }, 20_000);
});
