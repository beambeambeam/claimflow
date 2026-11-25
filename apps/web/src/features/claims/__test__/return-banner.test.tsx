import { screen, within } from "@testing-library/react";
import { http, HttpResponse } from "msw";
import { setupServer } from "msw/node";
import { beforeEach, describe, expect, it } from "vite-plus/test";

import { apiUrl, renderAt, serveApi } from "#/__test__/support";

import { claim, requester } from "./fixtures";

// The first render of a route loads its code chunks, which is slow on a busy machine.
const slow = { timeout: 10_000 };

const claimId = "0192f1c4-7b2e-7c3a-9d41-3f0a6b8e2c11";

const reviewer = {
  id: "usr-reviewer",
  name: "Malee R.",
  position: "Head, Section A",
};

const entry = (id: number, comment: string, createdAt: string) => ({
  id,
  action: "RETURN" as const,
  fromStatus: "SUBMITTED" as const,
  toStatus: "RETURNED" as const,
  actor: reviewer,
  comment,
  createdAt,
});

const returned = claim({
  status: "RETURNED",
  returnCount: 2,
  allowedActions: [],
  history: [
    entry(2, "Please attach the receipt.", "2026-11-04T09:30:00Z"),
    entry(3, "The taxi receipt is still missing.", "2026-11-06T09:30:00Z"),
  ],
});

let current = returned;

const server = setupServer(
  http.get(`${apiUrl}/auth/me`, () => HttpResponse.json({ data: requester })),
  http.get(`${apiUrl}/claims/${claimId}`, () =>
    HttpResponse.json({ data: current })
  )
);

serveApi(server);

beforeEach(() => {
  current = returned;
});

describe("Claim Detail of a returned claim", () => {
  it("F06-TEST-14 (banner and count; Edit and Resubmit are #44): shows the latest return reason with who returned it and when, and the number of returns", async () => {
    renderAt(`/claims/${claimId}`);

    await screen.findByRole("heading", { name: "CLM-2026-000123" }, slow);

    screen.getByText("Returned by Malee R. on 6 Nov 2026");
    // In the banner and in the history.
    expect(
      screen.getAllByText("The taxi receipt is still missing.")
    ).toHaveLength(2);
    expect(screen.queryAllByText("Please attach the receipt.")).toHaveLength(1);
    within(screen.getByRole("region", { name: "Summary" })).getByText(
      "Returned 2 / 3"
    );
  }, 20_000);

  it("F06-TEST-14: shows the banner without a reason when the return entry has none", async () => {
    current = claim({
      status: "RETURNED",
      returnCount: 1,
      allowedActions: [],
      history: [{ ...entry(2, "", "2026-11-04T09:30:00Z"), comment: null }],
    });

    renderAt(`/claims/${claimId}`);

    await screen.findByRole("heading", { name: "CLM-2026-000123" }, slow);

    screen.getByText("Returned by Malee R. on 4 Nov 2026");
  }, 20_000);
});
