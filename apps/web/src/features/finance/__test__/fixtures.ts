import type {
  Claim,
  ClaimSummary,
  CurrentUser,
  Problem,
} from "#/lib/api/types.gen";

export const financeOfficer: CurrentUser = {
  id: "usr-finance",
  username: "fin.wanida",
  name: "Wanida",
  role: "finance",
  permissions: { review: ["list"], finance: ["list"], report: ["read"] },
};

export const reviewer: CurrentUser = {
  id: "usr-reviewer",
  username: "rev.malee",
  name: "Malee R.",
  role: "reviewer",
  permissions: { claim: ["create", "list"], review: ["list"] },
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
  title: "Vendor site travel",
  status: "PENDING_FINANCE",
  total: "15000.00",
  requester: {
    id: "usr-requester",
    name: "Anan K.",
    position: "Staff, Section A",
  },
  submittedAt: "2026-11-05T03:00:00Z",
  createdAt: "2026-11-05T03:00:00Z",
  ...overrides,
});

const actor = (id: string, name: string, position: string) => ({
  id,
  name,
  position,
});

const anan = actor("usr-requester", "Anan K.", "Staff, Section A");
const malee = actor("usr-reviewer", "Malee R.", "Head, Section A");

// A 15,000.00 claim that rev.malee approved, which waits for the final decision by Finance.
export const pendingClaim = (overrides: Partial<Claim> = {}): Claim => ({
  id: "0192f1c4-7b2e-7c3a-9d41-3f0a6b8e2c11",
  claimNo: "CLM-2026-000123",
  status: "PENDING_FINANCE",
  title: "Vendor site travel",
  purpose: "Flights to the vendor site",
  total: "15000.00",
  returnCount: 0,
  version: 1,
  requester: anan,
  assignedReviewer: malee,
  firstSubmittedAt: "2026-11-05T03:00:00Z",
  submittedAt: "2026-11-05T03:00:00Z",
  decidedAt: null,
  createdAt: "2026-11-05T03:00:00Z",
  updatedAt: "2026-11-06T03:00:00Z",
  items: [
    {
      lineNo: 1,
      category: "TRAVEL",
      description: "Return flights",
      expenseDate: "2026-11-03",
      amount: "15000.00",
      receiptRef: "TG-1",
    },
  ],
  history: [
    {
      id: 1,
      action: "SUBMIT",
      fromStatus: null,
      toStatus: "SUBMITTED",
      actor: anan,
      comment: null,
      createdAt: "2026-11-05T03:00:00Z",
    },
    {
      id: 2,
      action: "APPROVE",
      fromStatus: "SUBMITTED",
      toStatus: "PENDING_FINANCE",
      actor: malee,
      comment: null,
      createdAt: "2026-11-06T03:00:00Z",
    },
  ],
  allowedActions: ["final-approve", "final-reject"],
  ...overrides,
});
