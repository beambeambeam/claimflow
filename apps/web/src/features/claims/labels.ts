import type { ClaimAction, ExpenseCategory } from "#/lib/api/types.gen";

export const categoryLabels: Record<ExpenseCategory, string> = {
  TRAVEL: "Travel",
  ACCOMMODATION: "Accommodation",
  MEALS: "Meals",
  SUPPLIES: "Supplies",
  TRAINING: "Training",
  OTHER: "Other",
};

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
