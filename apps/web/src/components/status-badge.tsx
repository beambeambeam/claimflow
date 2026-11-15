import { Badge } from "@astryxdesign/core/Badge";

import type { ClaimStatus } from "#/lib/api/types.gen";

export const statusLabels: Record<ClaimStatus, string> = {
  DRAFT: "Draft",
  SUBMITTED: "Submitted",
  RETURNED: "Returned",
  PENDING_FINANCE: "Pending finance",
  APPROVED: "Approved",
  REJECTED: "Rejected",
  WITHDRAWN: "Withdrawn",
};

// SDS §7.2 colors the 3 statuses of Sprint 1 and leaves the rest neutral.
const variants: Partial<Record<ClaimStatus, "blue" | "green" | "red">> = {
  SUBMITTED: "blue",
  APPROVED: "green",
  REJECTED: "red",
};

export const StatusBadge = ({ status }: { status: ClaimStatus }) => (
  <Badge label={statusLabels[status]} variant={variants[status] ?? "neutral"} />
);
