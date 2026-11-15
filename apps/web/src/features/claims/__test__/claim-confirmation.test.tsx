import { cleanup, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it } from "vite-plus/test";

import { ClaimConfirmation } from "#/features/claims/components/claim-confirmation";
import type { Claim } from "#/lib/api/types.gen";

afterEach(cleanup);

const claimWithTotal = (total: string): Claim => ({
  id: "0192f1c4-7b2e-7c3a-9d41-3f0a6b8e2c11",
  claimNo: "CLM-2026-000123",
  status: "SUBMITTED",
  title: "Bangkok workshop travel",
  purpose: "Train fares and hotel",
  total,
  returnCount: 0,
  version: 0,
  requester: { id: "usr-1", name: "Anan K.", position: "Staff, Section A" },
  assignedReviewer: null,
  firstSubmittedAt: null,
  submittedAt: null,
  decidedAt: null,
  createdAt: "2026-11-05T03:00:00Z",
  updatedAt: "2026-11-05T03:00:00Z",
  items: [],
  history: [],
  allowedActions: [],
});

describe("Claim confirmation", () => {
  it("shows the total in baht with 2 decimals", () => {
    render(
      <ClaimConfirmation
        claim={claimWithTotal("1250.00")}
        onNewClaim={() => {}}
      />
    );

    screen.getByText("Total ฿1,250.00");
  });

  it("shows a total that is not an amount as received, not as ฿0.00", () => {
    render(
      <ClaimConfirmation claim={claimWithTotal("abc")} onNewClaim={() => {}} />
    );

    screen.getByText("Total ฿abc");
  });

  it("links to the claim detail page", () => {
    render(
      <ClaimConfirmation
        claim={claimWithTotal("1250.00")}
        onNewClaim={() => {}}
      />
    );

    expect(
      screen.getByRole("link", { name: "View claim" }).getAttribute("href")
    ).toBe("/claims/0192f1c4-7b2e-7c3a-9d41-3f0a6b8e2c11");
  });
});
