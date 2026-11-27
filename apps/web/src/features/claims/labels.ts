import type { ClaimAction } from "#/lib/api/types.gen";

// How the history timeline names what an actor did.
export const actionLabels: Record<ClaimAction, string> = {
  SUBMIT: "Submitted",
  APPROVE: "Approved",
  REJECT: "Rejected",
  RETURN: "Returned",
  RESUBMIT: "Resubmitted",
  WITHDRAW: "Withdrawn",
  FINAL_APPROVE: "Final approval",
  FINAL_REJECT: "Final rejection",
};
