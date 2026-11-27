import type { ExpenseCategory } from "#/lib/api/types.gen";

// The label of each expense category, in the order of the ExpenseCategory enum.
export const categoryLabels: Record<ExpenseCategory, string> = {
  TRAVEL: "Travel",
  ACCOMMODATION: "Accommodation",
  MEALS: "Meals",
  SUPPLIES: "Supplies",
  TRAINING: "Training",
  OTHER: "Other",
};
