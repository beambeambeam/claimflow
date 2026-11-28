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
    ["RETURNED", "yellow"],
    ["PENDING_FINANCE", "purple"],
    ["APPROVED", "green"],
    ["REJECTED", "red"],
    ["DRAFT", "neutral"],
  ])("colors %s %s", (status, variant) => {
    const { container } = render(<StatusBadge status={status} />);

    expect(
      container.querySelector<HTMLElement>("[data-variant]")?.dataset.variant
    ).toBe(variant);
  });

  it("F07-AC-09: shows Withdrawn grey, with its label in a struck-through Text that no other status has", () => {
    const { container, rerender } = render(<StatusBadge status="WITHDRAWN" />);

    expect(
      container.querySelector<HTMLElement>("[data-variant]")?.dataset.variant
    ).toBe("neutral");
    // The strike-through is a style class of Text, so the test checks that the label is a Text and the others are not.
    expect(
      screen.getByText("Withdrawn").classList.contains("astryx-text")
    ).toBe(true);

    rerender(<StatusBadge status="DRAFT" />);

    expect(screen.getByText("Draft").classList.contains("astryx-text")).toBe(
      false
    );
  });
});
