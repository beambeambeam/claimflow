import type {
  Claim,
  ClaimSummary,
  CurrentUser,
  Problem,
} from "#/lib/api/types.gen";

export const reviewer: CurrentUser = {
  id: "usr-reviewer",
  username: "rev.malee",
  name: "Malee R.",
  role: "reviewer",
  permissions: { claim: ["create", "list"], review: ["list"] },
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
  requestId: string,
  errors?: Problem["errors"]
): Problem => ({
  type: "about:blank",
  title: "Problem",
  status,
  code,
  instance: "/api/v1/claims",
  requestId,
  errors,
});

export const summary = (overrides: Partial<ClaimSummary>): ClaimSummary => ({
  id: "0192f1c4-7b2e-7c3a-9d41-3f0a6b8e2c11",
  claimNo: "CLM-2026-000123",
  title: "Bangkok workshop travel",
  status: "SUBMITTED",
  total: "1250.00",
  requester: {
    id: "usr-requester",
    name: "Anan K.",
    position: "Staff, Section A",
  },
  submittedAt: "2026-11-05T03:00:00Z",
  createdAt: "2026-11-05T03:00:00Z",
  ...overrides,
});

// A SUBMITTED claim assigned to the reviewer, which the reviewer may approve and reject.
export const claim = (overrides: Partial<Claim> = {}): Claim => ({
  id: "0192f1c4-7b2e-7c3a-9d41-3f0a6b8e2c11",
  claimNo: "CLM-2026-000123",
  status: "SUBMITTED",
  title: "Bangkok workshop travel",
  purpose: "Train fares and hotel for the workshop",
  total: "1250.00",
  returnCount: 0,
  version: 0,
  requester: {
    id: "usr-requester",
    name: "Anan K.",
    position: "Staff, Section A",
  },
  assignedReviewer: {
    id: "usr-reviewer",
    name: "Malee R.",
    position: "Head, Section A",
  },
  firstSubmittedAt: "2026-11-05T03:00:00Z",
  submittedAt: "2026-11-05T03:00:00Z",
  decidedAt: null,
  createdAt: "2026-11-05T03:00:00Z",
  updatedAt: "2026-11-05T03:00:00Z",
  items: [
    {
      lineNo: 1,
      category: "TRAVEL",
      description: "Train return",
      expenseDate: "2026-11-03",
      amount: "1250.00",
      receiptRef: "SRT-1",
    },
  ],
  history: [
    {
      id: 1,
      action: "SUBMIT",
      fromStatus: null,
      toStatus: "SUBMITTED",
      actor: {
        id: "usr-requester",
        name: "Anan K.",
        position: "Staff, Section A",
      },
      comment: null,
      createdAt: "2026-11-05T03:00:00Z",
    },
  ],
  allowedActions: ["approve", "reject"],
  ...overrides,
});
