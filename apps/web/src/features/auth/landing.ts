import type { CurrentUser } from "#/lib/api/types.gen";

// Landing pages from F-01 §2.4.
const landingPaths: Record<CurrentUser["role"], "/claims" | "/reviews"> = {
  requester: "/claims",
  reviewer: "/reviews",
  finance: "/reviews",
};

export const getLandingPath = (role: CurrentUser["role"]) => landingPaths[role];
