import { cleanup, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it } from "vite-plus/test";

import { StatusBadge } from "#/components/status-badge";
import type { ClaimStatus } from "#/lib/api/types.gen";

afterEach(cleanup);

describe("StatusBadge", () => {
  it.each<[ClaimStatus, string]>([
    ["DRAFT", "Draft"],
    ["SUBMITTED", "Submitted"],
    ["RETURNED", "Returned"],
    ["PENDING_FINANCE", "Pending finance"],
    ["APPROVED", "Approved"],
    ["REJECTED", "Rejected"],
    ["WITHDRAWN", "Withdrawn"],
  ])("labels %s as %s", (status, label) => {
    render(<StatusBadge status={status} />);

    screen.getByText(label);
  });

  it.each<[ClaimStatus, string]>([
    ["SUBMITTED", "blue"],
    ["APPROVED", "green"],
    ["REJECTED", "red"],
    ["DRAFT", "neutral"],
  ])("colors %s %s", (status, variant) => {
    const { container } = render(<StatusBadge status={status} />);

    expect(
      container.querySelector<HTMLElement>("[data-variant]")?.dataset.variant
    ).toBe(variant);
  });
});
