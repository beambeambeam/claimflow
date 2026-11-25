import { Badge } from "@astryxdesign/core/Badge";
import { Text } from "@astryxdesign/core/Text";

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

// The badge of each status (SDS §7.2). A status without an entry is neutral and not struck through.
const styles: Partial<
  Record<
    ClaimStatus,
    {
      variant: "neutral" | "blue" | "yellow" | "green" | "red";
      isStruck?: boolean;
    }
  >
> = {
  SUBMITTED: { variant: "blue" },
  RETURNED: { variant: "yellow" },
  APPROVED: { variant: "green" },
  REJECTED: { variant: "red" },
  WITHDRAWN: { variant: "neutral", isStruck: true },
};

export const StatusBadge = ({ status }: { status: ClaimStatus }) => (
  <Badge
    label={
      styles[status]?.isStruck === true ? (
        <Text hasStrikethrough type="inherit">
          {statusLabels[status]}
        </Text>
      ) : (
        statusLabels[status]
      )
    }
    variant={styles[status]?.variant ?? "neutral"}
  />
);
