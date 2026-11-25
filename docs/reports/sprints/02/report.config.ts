import type { SprintConfig } from "../../scripts/lib/sprint.ts";

// Sprint 2 report (Deliverable 2): what the shared scripts capture and list.
// The accepted Sprint 2 release is v0.1.1.0 = 3a91ea4 (release PR #109).
const config: SprintConfig = {
  board: "https://github.com/users/beambeambeam/projects/5",
  ciCheck: (commit) =>
    String.raw`CI check: the required checks (\texttt{code\_quality}, \texttt{test}, \texttt{api}, \texttt{build}, \texttt{image}) passed on ${commit} (push to \texttt{main}, CI run 45614889848); the \texttt{api} job runs the generate check and the 100\% coverage gate. pass`,
  contracts: "docs/sprints/sprint-2",
  docs: {
    "adr-0001": "docs/adr/0001-finance-officer-reviews-department-head.md",
    dod: "docs/sprints/sprint-2/definition-of-done.md",
    f05: "docs/sprints/sprint-2/F-05-drafts.md",
    f06: "docs/sprints/sprint-2/F-06-return-resubmit.md",
    f07: "docs/sprints/sprint-2/F-07-withdraw.md",
    features: "docs/features.md",
    srs: "docs/srs.md",
    "system-sds": "docs/system-sds.md",
    traceability: "docs/traceability.md",
  },
  parts: {
    "part-b-1-contracts": [
      ["f05", "F-05 contract"],
      ["f06", "F-06 contract"],
      ["f07", "F-07 contract"],
      ["dod", "Sprint 2 Definition of Done"],
    ],
    "part-d-1-requirements": [
      ["srs", "Software Requirements Specification"],
      ["features", "Feature inventory"],
    ],
    "part-d-2-design": [
      ["system-sds", "System-level SDS"],
      ["adr-0001", "ADR 0001"],
    ],
    "part-d-3-traceability": [
      ["traceability", "Requirements traceability matrix"],
    ],
  },
  // Chain 1: tickets #43, #44, #45 -> PR #108, author Kiatisakk, reviewer
  // beambeambeam. Chain 2: tickets #39 to #42 -> PR #104, author beambeambeam,
  // reviewer Kiatisakk. Release: PR #109, header only.
  reviews: [
    { anchor: null, group: "pr108", name: "pr108-header", pr: 108 },
    {
      anchor: "pullrequestreview-6053459711",
      group: "pr108",
      name: "pr108-changes-requested",
      pr: 108,
    },
    {
      anchor: "discussion_r4691685396",
      group: "pr108",
      name: "pr108-thread-returned-badge",
      pr: 108,
    },
    {
      anchor: "pullrequestreview-6054373300",
      group: "pr108",
      name: "pr108-approved",
      pr: 108,
    },
    { anchor: null, group: "pr104", name: "pr104-header", pr: 104 },
    {
      anchor: "pullrequestreview-6043651469",
      group: "pr104",
      name: "pr104-changes-requested",
      pr: 104,
    },
    {
      anchor: "discussion_r4683494358",
      group: "pr104",
      name: "pr104-thread-stale-evidence",
      pr: 104,
    },
    {
      anchor: "pullrequestreview-6043855256",
      group: "pr104",
      name: "pr104-approved",
      pr: 104,
    },
    { anchor: null, group: "pr109", name: "pr109-header", pr: 109 },
  ],
  release: "v0.1.1.0",
  implementation: {
    "F05-AC-01":
      "POST /claims?draft=true: Handler.CreateClaim, Service.Create with submit=false (internal/claim)",
    "F05-AC-02":
      "rules.go: item and total limits for drafts, dates skipped until submit",
    "F05-AC-03":
      "policy.go and Service.Create: BR-15 guard, no reviewer means ErrForbidden",
    "F05-AC-04":
      "policy.go: owner-only access to a DRAFT; allowedActions from Allowed",
    "F05-AC-05":
      "PUT /claims/{id}: Handler.UpdateClaim, edit.go (replace items, recompute total)",
    "F05-AC-06": "policy.go owner check; lifecycle.go BR-10 on a non-draft",
    "F05-AC-07":
      "Handler If-Match parsing (428, 400); version compare in Service",
    "F05-AC-08":
      "POST /claims/{id}/submit: Handler.SubmitClaim, Service DRAFT to SUBMITTED with SUBMIT history",
    "F05-AC-09":
      "rules.go: expense-date window counted from the submission time",
    "F05-AC-10": "Service submit reads the owner's current reviewer (BR-17)",
    "F05-AC-11":
      "policy.go: no submit without a reviewer; Service submit guard (BR-15)",
    "F05-AC-12":
      "lifecycle.go transitions (BR-03); conditional UPDATE on version",
    "F05-AC-13": "Service: status update and history insert in one transaction",
    "F05-AC-14":
      "DELETE /claims/{id}: Handler.DeleteClaim, Service delete for a DRAFT of the owner",
    "F05-AC-15":
      "migration 00002_claims.sql: history foreign key and append-only trigger",
    "F05-AC-16": "web: new-claim-page.tsx, claim-form.tsx (Save as draft)",
    "F05-AC-17": "web: claim-detail-page.tsx, claim-owner-actions.tsx",
    "F05-AC-18": "web: edit-claim-page.tsx and the claims/$claimId_.edit route",
    "F05-AC-19": "web: my-claims-page.tsx, status-badge.tsx",
    "F05-AC-20": "web/e2e/f05-drafts.spec.ts over the API and web above",
    "F06-AC-01":
      "POST /claims/{id}/return: Handler.ReturnClaim, Service.Act with ActionReturn",
    "F06-AC-02":
      "rules.go: reason length shared with Reject (MinReasonLength, MaxCommentLength)",
    "F06-AC-03":
      "policy.go: assigned-reviewer relation (BR-01, BR-02, ADR 0001)",
    "F06-AC-04": "lifecycle.go transitions (BR-03); version compare (BR-13)",
    "F06-AC-05":
      "policy.go: MaxReturns = 3 hides return (BR-14); Service.Act refuses it",
    "F06-AC-06": "list scopes in Service.List; policy for the reviewer view",
    "F06-AC-07":
      "Service submit from RETURNED keeps the reviewer and firstSubmittedAt",
    "F06-AC-08":
      "rules.go: 90-day window counted from firstSubmittedAt (BR-07)",
    "F06-AC-09": "edit.go and policy.go: owner edits a RETURNED claim",
    "F06-AC-10": "Service: status update and history insert in one transaction",
    "F06-AC-11": "web: decision-panel.tsx (Return dialog, Returned n / 3)",
    "F06-AC-12":
      "web: claim-detail-page.tsx banner, claim-owner-actions.tsx Resubmit",
    "F06-AC-13":
      "web/e2e/f06-return-resubmit.spec.ts over the API and web above",
    "F07-AC-01":
      "POST /claims/{id}/withdraw: Handler.WithdrawClaim, Service.Act with ActionWithdraw",
    "F07-AC-02":
      "openapi.yaml ClaimWithdrawal.comment (x-max-length 500); validate.go",
    "F07-AC-03":
      "lifecycle.go transitions: only SUBMITTED and RETURNED reach WITHDRAWN (BR-16)",
    "F07-AC-04": "policy.go: ownership check for withdraw",
    "F07-AC-05":
      "Handler If-Match parsing (428, 400); version compare in Service",
    "F07-AC-06":
      "lifecycle.go: WITHDRAWN has no outgoing transition; policy gives []",
    "F07-AC-07": "Service: status update and history insert in one transaction",
    "F07-AC-08":
      "Service.List review scope excludes WITHDRAWN; Get still allowed",
    "F07-AC-09":
      "web: claim-owner-actions.tsx (Withdraw dialog), status-badge.tsx",
    "F07-AC-10": "web/e2e/f07-withdraw.spec.ts over the API and web above",
  },

  tickets: [],
  // Contributions: everything after the Sprint 1 release (ac5bce5, 2026-11-15).
  since: "2026-11-16",
};

export default config;
