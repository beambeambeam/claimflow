# Requirements Traceability Matrix

- **Product:** ClaimFlow — Department Expense Reimbursement System
- **Baseline:** `docs/srs.md` v0.2 · Features: `docs/features.md` v1.0
- **Last updated:** 2026-11-01

There is one row per SRS requirement. Each row traces requirement → feature or system responsibility → sprint → acceptance criteria → test → result.

How to read and update the columns:

- **AC / Planned test:** filled from each feature contract. Sprint 1 IDs come from the Sprint 1 contracts (#6). Rows for Sprints 2–3 and Lab 5 stay `planned` until their contracts are written.
- **Actual test:** the test file and name that runs, e.g. `internal/claim/lifecycle_test.go › TestApprove_SelfApproval`.
- **Result / Version:** `pass`, `fail`, or `manual`, plus the release tag or commit it was run on.
- **Ownership:** when a requirement is enforced by several features, the row lists all of them, and each feature's contract carries its own AC for its own paths.

---

## 1. Functional requirements

| Requirement | Feature/responsibility | Sprint | AC | Planned test | Actual test | Result | Version |
| --- | --- | --- | --- | --- | --- | --- | --- |
| FR-01 Log in, log out, current user | F-01 | 1 | TBD (#6) | TBD (#6) | — | — | — |
| FR-02 Create and submit a claim with line items | F-02 | 1 | TBD (#6) | TBD (#6) | — | — | — |
| FR-03 Server calculates the claim total | F-02 | 1 | TBD (#6) | TBD (#6) | — | — | — |
| FR-04 List and view own claims with history | F-03 | 1 | TBD (#6) | TBD (#6) | — | — | — |
| FR-05 Reviewer queue and claim detail | F-04 | 1 | TBD (#6) | TBD (#6) | — | — | — |
| FR-06 Approve or reject (reason on reject) | F-04 | 1 | TBD (#6) | TBD (#6) | — | — | — |
| FR-07 History entry for every status change | F-04 (S1); F-06, F-07 (S2); F-08 (S3) | 1 / 2 / 3 | TBD (#6) | TBD (#6) | — | — | — |
| FR-08 Drafts: save, edit, submit, delete | F-05 | 2 | planned | planned | — | — | — |
| FR-09 Return with reason; show return count | F-06 | 2 | planned | planned | — | — | — |
| FR-10 Edit and resubmit a returned claim | F-06 | 2 | planned | planned | — | — | — |
| FR-11 Withdraw while `Submitted` or `Returned` | F-07 | 2 | planned | planned | — | — | — |
| FR-12 Two-stage approval for high-value claims | F-08 | 3 | planned | planned | — | — | — |
| FR-13 Filter and search claims | F-09 | 3 | planned | planned | — | — | — |
| FR-14 Decision summaries | F-10 | 3 | planned | planned | — | — | — |

## 2. Business rules

| Requirement | Feature/responsibility | Sprint | AC | Planned test | Actual test | Result | Version |
| --- | --- | --- | --- | --- | --- | --- | --- |
| BR-01 No self-approval | F-04 (S1); F-06 return (S2); F-08 final decision (S3) | 1 / 2 / 3 | TBD (#6) for S1; planned | TBD (#6) for S1; planned | — | — | — |
| BR-02 Assigned reviewer only; Finance final decision only on `PendingFinance` | F-04 (S1); F-06 return (S2); F-08 (S3) | 1 / 2 / 3 | TBD (#6) for S1; planned | TBD (#6) for S1; planned | — | — | — |
| BR-03 Valid transitions only | SR-04; F-04 (S1); F-05, F-06, F-07 (S2); F-08 (S3) | 1 / 2 / 3 | TBD (#6) for S1; planned | TBD (#6) for S1; planned | — | — | — |
| BR-04 Reason required for reject, return, final reject | F-04 reject (S1); F-06 return (S2); F-08 final reject (S3) | 1 / 2 / 3 | TBD (#6) for S1; planned | TBD (#6) for S1; planned | — | — | — |
| BR-05 1–20 line items; amount 0.01–20,000.00 | F-02 | 1 | TBD (#6) | TBD (#6) | — | — | — |
| BR-06 Claim total ≤ 50,000.00 | F-02 | 1 | TBD (#6) | TBD (#6) | — | — | — |
| BR-07 Expense date not in future; ≤ 90 days before first submission | F-02 (S1); F-06 resubmit (S2) | 1 / 2 | TBD (#6) for S1; planned | TBD (#6) for S1; planned | — | — | — |
| BR-08 Server-authoritative totals; items change only in `Draft`/`Returned` | F-02 | 1 | TBD (#6) | TBD (#6) | — | — | — |
| BR-09 Two-stage threshold > 10,000.00; Finance reviewer's approval is final | F-08 | 3 | planned | planned | — | — | — |
| BR-10 Edit only in `Draft`/`Returned`; delete only `Draft`; owner only | F-05 (draft); F-06 (returned) | 2 | planned | planned | — | — | — |
| BR-11 History is immutable | SR-04; F-04 | 1 | TBD (#6) | TBD (#6) | — | — | — |
| BR-12 Categories; Meals ≤ 1,500.00 | F-02 | 1 | TBD (#6) | TBD (#6) | — | — | — |
| BR-13 Concurrent decisions: at most one succeeds | SR-04; F-04 | 1 | TBD (#6) | TBD (#6) | — | — | — |
| BR-14 Return at most 3 times | F-06 | 2 | planned | planned | — | — | — |
| BR-15 No assigned reviewer, no claims (incl. drafts) | F-02 (S1); F-05 drafts (S2) | 1 / 2 | TBD (#6) for S1; planned | TBD (#6) for S1; planned | — | — | — |
| BR-16 Withdraw only from `Submitted`/`Returned` | F-07 | 2 | planned | planned | — | — | — |
| BR-17 Reviewer fixed at first submission | F-02 (S1); F-06 resubmit (S2) | 1 / 2 | TBD (#6) for S1; planned | TBD (#6) for S1; planned | — | — | — |

## 3. Non-functional requirements

"Established" is the sprint that builds the capability; "verified" is the sprint whose tests or checks prove it.

| Requirement | Feature/responsibility | Sprint (established → verified) | AC | Planned test | Actual test | Result | Version |
| --- | --- | --- | --- | --- | --- | --- | --- |
| NFR-01 Passwords stored only as bcrypt/argon2 hashes | F-01 | 1 → 1 | TBD (#6) | TBD (#6) | — | — | — |
| NFR-02 401/403 enforced on every protected endpoint | SR-01; every feature with endpoints | 1 → 1, 2, 3 | TBD (#6) for S1; planned | TBD (#6) for S1; planned | — | — | — |
| NFR-03 Session ≤ 8 h; logout invalidates immediately | F-01 | 1 → 1 | TBD (#6) | TBD (#6) | — | — | — |
| NFR-04 No secrets in Git; config from env | SR-02 | 1 → 1 (local scan); Lab 5 (cloud secrets) | TBD (#6) for S1; planned (Lab 5) | TBD (#6) for S1; planned (Lab 5) | — | — | — |
| NFR-05 Data survives restart and redeploy | SR-03 | 1 → 1 (local restart); Lab 5 (GCP redeploy) | TBD (#6) for S1; planned (Lab 5) | TBD (#6) for S1; planned (Lab 5) | — | — | — |
| NFR-06 Status change and history in one transaction | SR-04; F-04 | 1 → 1, 2, 3 | TBD (#6) for S1; planned | TBD (#6) for S1; planned | — | — | — |
| NFR-07 95% of requests < 1 s with ≥ 200 claims | SR-05; F-03, F-04, F-09, F-10 | 3 → 3 | planned | planned | — | — | — |
| NFR-08 Field-level validation messages naming the rule | F-02 (S1); F-05, F-06 (S2) | 1 → 1, 2 | TBD (#6) for S1; planned | TBD (#6) for S1; planned | — | — | — |
| NFR-09 Every BR tested; 100% Go coverage; CI on PRs | SR-06 | 1 → 1, 2, 3 | TBD (#6) for S1; planned | CI `api` job (`go-test-coverage`) | — | — | — |
| NFR-10 One-command Docker run | SR-07 | Lab 5 → Lab 5 | planned (Lab 5) | planned (Lab 5) | — | — | — |

## 4. Coverage

Checked against `docs/srs.md` v0.2 on 2026-11-01.

| Type | In SRS | Traced here | Mapped to a feature or SR | Uncovered |
| --- | --- | --- | --- | --- |
| Functional (FR) | 14 | 14 | 14 | 0 |
| Business rules (BR) | 17 | 17 | 17 | 0 |
| Non-functional (NFR) | 10 | 10 | 10 | 0 |
| **Total** | **41** | **41** | **41** | **0** |

Reverse check: every feature F-01–F-10 and every responsibility SR-01–SR-07 traces back to at least one requirement (see `docs/features.md`).

**Uncovered requirements:** none.

**Verified outside a sprint deliverable** (by design, not gaps):

- NFR-04 (cloud half), NFR-05 (GCP redeploy), and NFR-10 are verified in **Lab 5** (due 15 Nov).

## 5. Change log

Record every approved change to requirements, features, or their mapping. Feature and SR IDs are frozen when this document is first merged; a removed ID is marked as retired, never reused.

| Date | Change         | Affected IDs | Reason | Approved by |
| ---- | -------------- | ------------ | ------ | ----------- |
| —    | No changes yet | —            | —      | —           |
