# Requirements Traceability Matrix

- **Product:** ClaimFlow – Department Expense Reimbursement System
- **Baseline:** `docs/srs.md` v0.2 · Features: `docs/features.md` v1.0 · Design: `docs/system-sds.md` v0.8 · Sprint 1 contracts: `docs/sprints/sprint-1/`
- **Last updated:** 2026-11-05

There is one row per SRS requirement. Each row traces requirement → feature or system responsibility → sprint → acceptance criteria → test → result. The design sections for each feature and responsibility are in section 4 of `docs/features.md`.

How to read and update the columns:

- **AC / Planned test:** filled from each feature contract. Sprint 1 IDs come from the Sprint 1 contracts in `docs/sprints/sprint-1/` (#6). `F02-AC-03` is AC-03 in the F-02 contract; `F02-TEST-04` is its TEST-04. Rows for Sprints 2–3 and Lab 5 stay `planned` until their contracts are written.
- **Actual test:** the test file and name that runs, e.g. `internal/claim/service_test.go › TestAct_SelfApproval`.
- **Result / Version:** `pass`, `fail`, or `manual`, plus the release tag or commit it was run on.
- **Ownership:** when a requirement is enforced by several features, the row lists all of them, and each feature's contract carries its own AC for its own paths.

---

## 1. Functional requirements

| Requirement | Feature/responsibility | Sprint | AC | Planned test | Actual test | Result | Version |
| --- | --- | --- | --- | --- | --- | --- | --- |
| FR-01 Log in, log out, current user | F-01 | 1 | F01-AC-01–04, 06, 07, 18–20 | F01-TEST-01–05, 07, 08, 20–22, 24 | – | – | – |
| FR-02 Create and submit a claim with line items | F-02 | 1 | F02-AC-01, 02, 12, 13 | F02-TEST-01–03, 18, 19, 21 | – | – | – |
| FR-03 Server calculates the claim total | F-02 (S1); F-05, F-06 (S2) | 1 / 2 | F02-AC-01, 08 (S1); planned | F02-TEST-01, 02, 13, 21 (S1); planned | – | – | – |
| FR-04 List and view own claims with history | F-03 | 1 | F03-AC-01–03, 05, 07, 08, 10, 11 | F03-TEST-01–03, 05, 07, 08, 10–12 | – | – | – |
| FR-05 Reviewer queue and claim detail | F-04 | 1 | F04-AC-01–03, 17, 19 | F04-TEST-01–03, 20, 22 | – | – | – |
| FR-06 Approve or reject (reason on reject) | F-04 | 1 | F04-AC-04, 05, 10, 18, 20 | F04-TEST-04, 05, 13, 21, 23, 24 | – | – | – |
| FR-07 History entry for every status change | F-02, F-04 (S1); F-05, F-06, F-07 (S2); F-08 (S3) | 1 / 2 / 3 | F02-AC-01; F04-AC-04, 05, 20 (S1); planned | F02-TEST-01, 02, 21; F04-TEST-04, 05, 23, 24 (S1); planned | – | – | – |
| FR-08 Drafts: save, edit, submit, delete | F-05 | 2 | planned | planned | – | – | – |
| FR-09 Return with reason; show return count | F-06 | 2 | planned | planned | – | – | – |
| FR-10 Edit and resubmit a returned claim | F-06 | 2 | planned | planned | – | – | – |
| FR-11 Withdraw while `Submitted` or `Returned` | F-07 | 2 | planned | planned | – | – | – |
| FR-12 Two-stage approval for high-value claims | F-08 | 3 | planned | planned | – | – | – |
| FR-13 Filter and search claims | F-09 | 3 | planned | planned | – | – | – |
| FR-14 Decision summaries | F-10 | 3 | planned | planned | – | – | – |

## 2. Business rules

| Requirement | Feature/responsibility | Sprint | AC | Planned test | Actual test | Result | Version |
| --- | --- | --- | --- | --- | --- | --- | --- |
| BR-01 No self-approval | F-04 (S1); F-06 return (S2); F-08 final decision (S3) | 1 / 2 / 3 | F04-AC-07 (S1); planned | F04-TEST-08–10 (S1); planned | – | – | – |
| BR-02 Assigned reviewer only; Finance final decision only on `PendingFinance` | F-04 (S1); F-06 return (S2); F-08 (S3) | 1 / 2 / 3 | F04-AC-08–10 (S1); planned | F04-TEST-11–13 (S1); planned | – | – | – |
| BR-03 Valid transitions only | SR-04; F-04 (S1); F-05, F-06, F-07 (S2); F-08 (S3) | 1 / 2 / 3 | F04-AC-11 (S1); planned | F04-TEST-14 (S1); planned | – | – | – |
| BR-04 Reason required for reject, return, final reject | F-04 reject (S1); F-06 return (S2); F-08 final reject (S3) | 1 / 2 / 3 | F04-AC-05, 06 (S1); planned | F04-TEST-05–07, 24 (S1); planned | – | – | – |
| BR-05 1–20 line items; amount 0.01–20,000.00 | F-02 (S1); F-05, F-06 edit (S2) | 1 / 2 | F02-AC-03, 04 (S1); planned | F02-TEST-04–07 (S1); planned | – | – | – |
| BR-06 Claim total ≤ 50,000.00 | F-02 (S1); F-05, F-06 edit (S2) | 1 / 2 | F02-AC-05 (S1); planned | F02-TEST-08 (S1); planned | – | – | – |
| BR-07 Expense date not in future; ≤ 90 days before first submission | F-02 (S1); F-05 draft submit, F-06 resubmit (S2) | 1 / 2 | F02-AC-07 (S1); planned | F02-TEST-11, 12 (S1); planned | – | – | – |
| BR-08 Server-authoritative totals; items change only in `Draft`/`Returned` | F-02 (S1); F-05, F-06 edit (S2) | 1 / 2 | F02-AC-08 (S1); planned | F02-TEST-13 (S1); planned | – | – | – |
| BR-09 Two-stage threshold > 10,000.00; Finance reviewer's approval is final | F-08 | 3 | planned | planned | – | – | – |
| BR-10 Edit only in `Draft`/`Returned`; delete only `Draft`; owner only | F-05 (draft); F-06 (returned) | 2 | planned | planned | – | – | – |
| BR-11 History is immutable | SR-04; F-04 | 1 | F04-AC-15 | F04-TEST-18 | – | – | – |
| BR-12 Categories; Meals ≤ 1,500.00 | F-02 (S1); F-05, F-06 edit (S2) | 1 / 2 | F02-AC-06 (S1); planned | F02-TEST-09, 10 (S1); planned | – | – | – |
| BR-13 Concurrent decisions: at most one succeeds | SR-04; F-04 | 1 | F04-AC-12, 13 | F04-TEST-15, 16 | – | – | – |
| BR-14 Return at most 3 times | F-06 | 2 | planned | planned | – | – | – |
| BR-15 No assigned reviewer, no claims (incl. drafts) | F-02 (S1); F-05 drafts (S2) | 1 / 2 | F01-AC-20; F02-AC-09, 14 (S1); planned | F01-TEST-22; F02-TEST-14, 15, 20 (S1); planned | – | – | – |
| BR-16 Withdraw only from `Submitted`/`Returned` | F-07 | 2 | planned | planned | – | – | – |
| BR-17 Reviewer fixed at first submission | F-02 (S1); F-05 draft submit, F-06 resubmit (S2) | 1 / 2 | F02-AC-01, 10 (S1); planned | F02-TEST-01, 02, 16, 21 (S1); planned | – | – | – |

## 3. Non-functional requirements

"Established" is the sprint that builds the capability; "verified" is the sprint whose tests or checks prove it.

| Requirement | Feature/responsibility | Sprint (established → verified) | AC | Planned test | Actual test | Result | Version |
| --- | --- | --- | --- | --- | --- | --- | --- |
| NFR-01 Passwords stored only as bcrypt/argon2 hashes | F-01 | 1 → 1 | F01-AC-09, 10 | F01-TEST-10, 11 | – | – | – |
| NFR-02 401/403 enforced on every protected endpoint | SR-01; every feature with endpoints | 1 → 1, 2, 3 | F01-AC-03, 11–14; F03-AC-02, 04, 06, 09; F04-AC-02, 08, 09 (S1); planned | F01-TEST-04, 12–15; F03-TEST-02, 04, 06, 09; F04-TEST-02, 11, 12 (S1); planned | – | – | – |
| NFR-03 Session ≤ 8 h; logout invalidates immediately | F-01 | 1 → 1 | F01-AC-05, 07, 08, 21 | F01-TEST-06, 08, 09, 23, 24 | – | – | – |
| NFR-04 No secrets in Git; config from env | SR-02 | 1 → 1 (local scan); Lab 5 (cloud secrets) | F01-AC-16 (S1); planned (Lab 5) | F01-TEST-17, 18 (S1); planned (Lab 5) | – | – | – |
| NFR-05 Data survives restart and redeploy | SR-03 | 1 → 1 (local restart); Lab 5 (GCP redeploy) | F04-AC-16 (S1); planned (Lab 5) | F04-TEST-19 (S1); planned (Lab 5) | – | – | – |
| NFR-06 Status change and history in one transaction | SR-04; F-04 | 1 → 1, 2, 3 | F02-AC-02; F04-AC-04, 14 (S1); planned | F02-TEST-03; F04-TEST-04, 17 (S1); planned | – | – | – |
| NFR-07 95% of requests < 1 s with ≥ 200 claims | SR-05; F-03, F-04, F-09, F-10 | 3 → 3 | planned | planned | – | – | – |
| NFR-08 Field-level validation messages naming the rule | F-02, F-04 reject reason (S1); F-05, F-06 (S2); F-08 final reject reason (S3) | 1 → 1, 2, 3 | F02-AC-11–13; F04-AC-06, 18 (S1); planned | F02-TEST-17–19, 21; F04-TEST-06, 07, 21, 24 (S1); planned | – | – | – |
| NFR-09 Every BR tested; 100% Go coverage; CI on PRs | SR-06 | 1 → 1, 2, 3 | F01-AC-17 (S1); planned | F01-TEST-19; CI `api` job (`go-test-coverage`) | – | – | – |
| NFR-10 One-command Docker run | SR-07 | Lab 5 → Lab 5 | planned (Lab 5) | planned (Lab 5) | – | – | – |

## 4. Coverage

Checked against `docs/srs.md` v0.2 on 2026-11-04.

| Type | In SRS | Traced here | Mapped to a feature or SR | Uncovered |
| --- | --- | --- | --- | --- |
| Functional (FR) | 14 | 14 | 14 | 0 |
| Business rules (BR) | 17 | 17 | 17 | 0 |
| Non-functional (NFR) | 10 | 10 | 10 | 0 |
| **Total** | **41** | **41** | **41** | **0** |

Reverse check: every feature F-01–F-10 and every responsibility SR-01–SR-07 traces back to at least one requirement (see `docs/features.md`).

**Uncovered requirements:** none.

**Sprint 1 contracts** (checked 2026-11-05): every Sprint 1 requirement above has at least one AC, and every one of the 66 AC has at least one of the 81 planned tests.

**Verified outside a sprint deliverable** (by design, not gaps):

- NFR-04 (cloud half), NFR-05 (GCP redeploy), and NFR-10 are verified in **Lab 5** (due 15 Nov).

## 5. Change log

Record every approved change to requirements, features, or their mapping. Feature and SR IDs are frozen when this document is first merged; a removed ID is marked as retired, never reused.

| Date | Change         | Affected IDs | Reason | Approved by |
| ---- | -------------- | ------------ | ------ | ----------- |
| –    | No changes yet | –            | –      | –           |
