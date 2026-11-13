import type { CurrentUser } from "#/lib/api/types.gen";

// Role labels from F-01 §2.2.
export const roleLabels: Record<CurrentUser["role"], string> = {
  requester: "Requester",
  reviewer: "Reviewer",
  finance: "Finance Officer",
};

// Access is gated by permission, not role name (F-01 §1.6).
export const can = (user: CurrentUser, resource: string, action: string) =>
  user.permissions[resource]?.includes(action) ?? false;
