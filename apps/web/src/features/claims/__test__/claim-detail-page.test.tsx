import { screen, within } from "@testing-library/react";
import { delay, http, HttpResponse } from "msw";
import { setupServer } from "msw/node";
import { beforeEach, describe, expect, it } from "vite-plus/test";

import { apiUrl, renderAt, serveApi } from "#/__test__/support";

import { claim, problem, requester } from "./fixtures";

// The first render of a route loads its code chunks, which is slow on a busy machine.
const slow = { timeout: 10_000 };

const claimId = "0192f1c4-7b2e-7c3a-9d41-3f0a6b8e2c11";

let claimResponse: () => Response = () => HttpResponse.json({ data: claim() });

// While set, the claim never answers, as when a request is still in flight.
let stalled = false;

const server = setupServer(
  http.get(`${apiUrl}/auth/me`, () => HttpResponse.json({ data: requester })),
  http.get(`${apiUrl}/claims/${claimId}`, async () => {
    if (stalled) {
      await delay("infinite");
    }

    return claimResponse();
  })
);

serveApi(server);

// Every claim field that must not appear when the claim is not shown.
const claimFields = [
  "CLM-2026-000123",
  "Bangkok workshop travel",
  "Train fares and hotel for the workshop",
  "Train return",
  "Hotel, one night",
  "SRT-1",
  "฿1,250.00",
  "Malee R.",
];

const expectNoClaimData = () => {
  for (const field of claimFields) {
    expect(screen.queryByText(field)).toBeNull();
  }
};

beforeEach(() => {
  stalled = false;
});

describe("Claim Detail", () => {
  it("F03-TEST-11: shows the header, summary, line items and the history oldest first", async () => {
    claimResponse = () => HttpResponse.json({ data: claim() });
    renderAt(`/claims/${claimId}`);

    await screen.findByRole("heading", { name: "CLM-2026-000123" }, slow);

    // Header
    screen.getByText("Bangkok workshop travel");
    const header = screen.getByRole("region", { name: "Claim" });
    within(header).getByText("Approved");
    within(header).getByText("฿1,250.00");

    // Summary
    const summary = screen.getByRole("region", { name: "Summary" });
    within(summary).getByText("Train fares and hotel for the workshop");
    within(summary).getByText("Anan K.");
    within(summary).getByText("Malee R.");
    within(summary).getByText("5 Nov 2026");

    // Line items
    const items = screen.getByRole("region", { name: "Line items" });
    expect(
      within(items)
        .getAllByRole("columnheader")
        .map((cell) => cell.textContent)
    ).toEqual([
      "#",
      "Category",
      "Description",
      "Expense date",
      "Amount",
      "Receipt ref",
    ]);
    const [, first, second] = within(items).getAllByRole("row");
    expect(
      within(first)
        .getAllByRole("cell")
        .map((cell) => cell.textContent)
    ).toEqual([
      "1",
      "Travel",
      "Train return",
      "3 Nov 2026",
      "฿250.00",
      "SRT-1",
    ]);
    expect(
      within(second)
        .getAllByRole("cell")
        .map((cell) => cell.textContent)
    ).toEqual([
      "2",
      "Accommodation",
      "Hotel, one night",
      "4 Nov 2026",
      "฿1,000.00",
      "SRT-2",
    ]);

    // History, oldest first
    const history = screen.getByRole("region", { name: "History" });
    const entries = within(history).getAllByRole("listitem");
    expect(entries).toHaveLength(2);
    within(entries[0]).getByText("Submitted");
    within(entries[0]).getByText("– → Submitted");
    within(entries[0]).getByText("Anan K.");
    within(entries[0]).getByText("5 Nov 2026, 10:00");
    within(entries[1]).getByText("Approved");
    within(entries[1]).getByText("Submitted → Approved");
    within(entries[1]).getByText("Malee R.");
    within(entries[1]).getByText("6 Nov 2026, 16:30");
    within(entries[1]).getByText("Receipts match.");

    // No action is allowed for the owner in Sprint 1.
    expect(within(screen.getByRole("main")).queryByRole("button")).toBeNull();
  }, 20_000);

  it("F03-TEST-11: shows a dash for a reviewer or submitted date the claim does not have", async () => {
    claimResponse = () =>
      HttpResponse.json({
        data: claim({ assignedReviewer: null, submittedAt: null }),
      });
    renderAt(`/claims/${claimId}`);

    await screen.findByRole("heading", { name: "CLM-2026-000123" }, slow);
    const summary = screen.getByRole("region", { name: "Summary" });
    expect(within(summary).getAllByText("–")).toHaveLength(2);
  }, 20_000);

  it("F03-TEST-11: shows a skeleton and no claim fields while the claim loads", async () => {
    stalled = true;
    renderAt(`/claims/${claimId}`);

    await screen.findByRole("table", {}, slow);
    expectNoClaimData();
    expect(screen.queryByRole("heading", { name: /^CLM-/u })).toBeNull();
  }, 20_000);

  it('F03-TEST-11: shows "You don\'t have access to this claim." on 403 and no claim fields', async () => {
    claimResponse = () =>
      HttpResponse.json(problem(403, "FORBIDDEN", "req-claim-403"), {
        status: 403,
      });
    renderAt(`/claims/${claimId}`);

    await screen.findByText("You don't have access to this claim.", {}, slow);
    expectNoClaimData();
    expect(screen.queryByRole("table")).toBeNull();
    expect(screen.queryByText("Claim not found.")).toBeNull();
  }, 20_000);

  it("F03-TEST-11: shows 'Claim not found.' on 404 and no claim fields", async () => {
    claimResponse = () =>
      HttpResponse.json(problem(404, "NOT_FOUND", "req-claim-404"), {
        status: 404,
      });
    renderAt(`/claims/${claimId}`);

    await screen.findByText("Claim not found.", {}, slow);
    expectNoClaimData();
    expect(screen.queryByRole("table")).toBeNull();
    expect(
      screen.queryByText("You don't have access to this claim.")
    ).toBeNull();
  }, 20_000);

  it("F03-TEST-11: shows an inline error with the request ID, and Try again fetches the claim again", async () => {
    claimResponse = () =>
      HttpResponse.json(problem(500, "INTERNAL", "req-claim-500"), {
        status: 500,
      });
    const { user } = renderAt(`/claims/${claimId}`);

    await screen.findByText("Request ID: req-claim-500", {}, slow);
    expectNoClaimData();
    expect(screen.queryByRole("table")).toBeNull();

    claimResponse = () => HttpResponse.json({ data: claim() });
    await user.click(screen.getByRole("button", { name: "Try again" }));

    await screen.findByRole("heading", { name: "CLM-2026-000123" });
    expect(screen.queryByText("Request ID: req-claim-500")).toBeNull();
  }, 20_000);

  it("F03-TEST-11: shows the error without a request ID when the network fails", async () => {
    claimResponse = () => HttpResponse.error();
    renderAt(`/claims/${claimId}`);

    await screen.findByText("Something went wrong.", {}, slow);
    screen.getByRole("button", { name: "Try again" });
    expect(screen.queryByText(/Request ID/u)).toBeNull();
    expectNoClaimData();
  }, 20_000);
});
