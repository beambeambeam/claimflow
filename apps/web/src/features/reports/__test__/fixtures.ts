import type { CurrentUser, Problem, Summary } from "#/lib/api/types.gen";

// The route permissions of a reviewer and of a Finance Officer include report:read (SDS §5.1).
export const reviewer: CurrentUser = {
  id: "usr-reviewer",
  username: "rev.malee",
  name: "Malee R.",
  role: "reviewer",
  permissions: {
    claim: ["create", "list"],
    review: ["list"],
    report: ["read"],
  },
};

export const requester: CurrentUser = {
  id: "usr-requester",
  username: "req.anan",
  name: "Anan K.",
  role: "requester",
  permissions: { claim: ["create", "list"] },
};

export const problem = (
  status: number,
  code: Problem["code"],
  requestId: string
): Problem => ({
  type: "about:blank",
  title: "Problem",
  status,
  code,
  instance: "/api/v1/reports/summary",
  requestId,
});

// Two months and three categories. October has no ACCOMMODATION, and November has no MEALS.
export const twoMonths: Summary = {
  countsByStatus: {
    SUBMITTED: 4,
    RETURNED: 1,
    PENDING_FINANCE: 2,
    APPROVED: 12,
    REJECTED: 3,
    WITHDRAWN: 1,
  },
  approvedTotals: [
    { month: "2026-10", category: "TRAVEL", total: "18250.00" },
    { month: "2026-10", category: "MEALS", total: "1200.00" },
    { month: "2026-11", category: "TRAVEL", total: "300.50" },
    { month: "2026-11", category: "ACCOMMODATION", total: "5000.00" },
  ],
  averageDecisionHours: 26.4,
};

// Claims in scope, but no decision in the range.
export const noDecisions: Summary = {
  countsByStatus: { ...twoMonths.countsByStatus },
  approvedTotals: [],
  averageDecisionHours: null,
};
