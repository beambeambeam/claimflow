# Software Requirements Specification (SRS)

- **Product:** ClaimFlow — Department Expense Reimbursement System
- **Course:** CPE 334 Group Project — Theme 5: Workflow and Approval System
- **Team:** 1005, 1045
- **Version:** 0.2 (draft for Deliverable 1)
- **Last updated:** 2026-10-31

Domain terms (Claim, Requester, Assigned Reviewer, Return, Withdraw, High-value Claim, …) are defined in `GLOSSARY.md`.

---

## 1. Purpose and context

A university department reimburses staff for work expenses they have already paid: travel, accommodation, meals, supplies, and training. Today claims are submitted on paper and approvals are tracked by email. As a result, claim status is unclear, decisions cannot be audited, and people can approve their own claims.

ClaimFlow works as follows:

- A **requester** submits a reimbursement claim with line items.
- The requester's **assigned reviewer** in the reporting line approves, rejects, or returns the claim with a reason.
- Every decision is kept in an append-only history.
- **High-value claims** need a final decision from a Finance Officer.

This document is the product baseline. Feature-level detail lives in each feature contract under `docs/sprints/`.

## 2. Actors

| ID | Actor | Responsibilities |
| --- | --- | --- |
| A-01 | Requester | A staff member who submits claims for reimbursement and tracks their own claims. |
| A-02 | Reviewer | A Section Head or the Department Head. Decides on claims from the people they are assigned to review. A Reviewer can also submit their own claims, which go to the next person in the reporting line. |
| A-03 | Finance Officer | Acts as the Department Head's assigned reviewer (Sprint 1). Makes the final decision on high-value claims (Sprint 3). Sees department-wide summaries (Sprint 3). Cannot submit claims. |
| A-04 | Anonymous visitor | Not logged in. Can only reach the login page. |

**Reporting line:**

```
Staff ──► Section Head ──► Department Head ──► Finance Officer
```

Each arrow points to that person's assigned reviewer. Section Head and Department Head are positions held by users with the Reviewer role, not separate roles. Each account has exactly one role (D-02).

## 3. Scope

### 3.1 In scope

- Authentication (login, logout, current user) and role- and ownership-based authorization enforced on the server.
- Reimbursement claims with one or more line items: create, submit, view, save as draft, edit, resubmit, withdraw, and delete drafts.
- Reviewer decisions: approve, reject, and return, each with a recorded history entry. A claim can be returned at most 3 times.
- Enforced claim lifecycle (Section 5).
- Two-stage approval for high-value claims (Sprint 3).
- Claim filtering/search and decision summaries (Sprint 3).
- Persistent database with seeded demo data; Docker packaging and GCP deployment (Lab 5).

### 3.2 Exclusions (out of scope)

| ID | Excluded item | Reason |
| --- | --- | --- |
| EX-01 | Pre-approval requests (e.g. asking permission before travelling) | Only reimbursement of expenses already paid is supported (D-01). |
| EX-02 | Real payment, bank transfer, or payroll integration | Course rule: no real payment gateway. `Approved` is the end of the workflow. |
| EX-03 | Email, SMS, or push notifications | No external messaging services. Status is visible in the app only. |
| EX-04 | Self-registration, password recovery, email verification | Not required; seeded accounts are used. |
| EX-05 | Receipt file upload and OCR | A receipt reference number is entered instead (D-09). |
| EX-06 | Multiple currencies and tax calculation | All amounts are in THB. |
| EX-07 | Budget tracking per project or cost centre | Separate product concern. |
| EX-08 | Account administration UI (create users, change roles, change the reporting line) | Managed through seed data/migrations. |
| EX-09 | Detecting collusion between reviewers | Outside the department's stated needs. |
| EX-10 | Claims by Finance Officers | Finance Officers have no assigned reviewer (D-06). |

## 4. Assumptions

| ID | Assumption |
| --- | --- |
| AS-01 | Users access the system through a modern desktop browser. Mobile layout is best-effort. |
| AS-02 | User accounts, roles, and the reporting line (each member's assigned reviewer) are provided as seed data. |
| AS-03 | A member's assigned reviewer is never the member themselves. |
| AS-04 | The department has one Department Head and at least one Finance Officer. |
| AS-05 | Synthetic data only; no real personal or financial data is stored. |
| AS-06 | The system uses a single timezone, Asia/Bangkok (UTC+7), for dates. |

## 5. Claim lifecycle

```
  (new) ──save draft (S2)──► Draft ──delete (S2)──► (removed)
    │                          │
    │ submit (S1)              │ submit (S2)
    ▼                          ▼
              Submitted ◄────────────── resubmit (S2) ──┐
      ┌──────────┼──────────────┬───────────┐          │
      │ approve  │ approve      │ reject    │ return   │
      │          │ high-value,  │           │ (S2,     │
      │          │ reviewer is  │           │  max 3)  │
      │          │ not Finance  │           ▼          │
      │          │ (S3)         │        Returned ─────┘
      ▼          ▼              ▼
  Approved ◄─ PendingFinance ─► Rejected
         final approve   final reject
            (S3)            (S3)

  Submitted, Returned ──withdraw (S2)──► Withdrawn
```

| From | Action | To | Who | Sprint |
| --- | --- | --- | --- | --- |
| (new) | submit | Submitted | Requester (owner) | 1 |
| Submitted | approve | Approved | Assigned reviewer | 1 |
| Submitted | reject | Rejected | Assigned reviewer | 1 |
| (new) | save draft | Draft | Requester (owner) | 2 |
| Draft | submit | Submitted | Owner | 2 |
| Draft | delete | (removed) | Owner | 2 |
| Submitted | return | Returned | Assigned reviewer, while fewer than 3 returns | 2 |
| Returned | resubmit | Submitted | Owner | 2 |
| Submitted, Returned | withdraw | Withdrawn | Owner | 2 |
| Submitted | approve (high-value, reviewer is not a Finance Officer) | PendingFinance | Assigned reviewer | 3 |
| PendingFinance | final approve / final reject | Approved / Rejected | Finance Officer | 3 |

Rules for the lifecycle:

- **Terminal states:** `Approved`, `Rejected`, and `Withdrawn`. No transition leaves a terminal state.
- **Sprints 1–2:** before the two-stage rule exists, `approve` always leads to `Approved`.
- **Invalid transitions:** any transition not in the table is rejected (BR-03).

## 6. Functional requirements

Priority: **M** = must, **S** = should.

| ID | Requirement | Actor | Priority | Sprint |
| --- | --- | --- | --- | --- |
| FR-01 | A user can log in with username and password and log out. The system can return the current user's identity and role. | All | M | 1 |
| FR-02 | A requester can create and submit a claim containing a title, purpose, and one or more line items (category, description, expense date, amount, receipt reference). On submission the claim is assigned to the requester's assigned reviewer. | A-01, A-02 | M | 1 |
| FR-03 | The system calculates a claim's total on the server from its line items. | System | M | 1 |
| FR-04 | A requester can list their own claims with status and total, and view any one of them in detail, including its history. | A-01, A-02 | M | 1 |
| FR-05 | A Reviewer or Finance Officer can list the claims assigned to them that are awaiting a decision, and view each one in detail, including its full history. | A-02, A-03 | M | 1 |
| FR-06 | An assigned reviewer can approve or reject a `Submitted` claim. A rejection requires a reason. | A-02, A-03 | M | 1 |
| FR-07 | The system records a history entry for every status change: previous status, new status, actor, timestamp, and comment. | System | M | 1 |
| FR-08 | A requester can save a claim as a Draft, edit it, submit it later, or delete it. | A-01, A-02 | S | 2 |
| FR-09 | An assigned reviewer can return a `Submitted` claim with a required reason. The reviewer sees how many times the claim has already been returned (e.g. "2 / 3"). | A-02, A-03 | M | 2 |
| FR-10 | A requester can edit a `Returned` claim and resubmit it to the same assigned reviewer. History keeps every earlier decision. | A-01, A-02 | M | 2 |
| FR-11 | A requester can withdraw their own claim while it is `Submitted` or `Returned`. | A-01, A-02 | M | 2 |
| FR-12 | A high-value claim approved by an assigned reviewer who is not a Finance Officer moves to `PendingFinance`. A Finance Officer then approves or rejects it. | A-02, A-03 | M | 3 |
| FR-13 | Users can filter the claims they are allowed to see by status, category, and date range, and search them by title. | All | S | 3 |
| FR-14 | Decision summaries: a Reviewer sees counts by status, approved total by category and month, and average decision time for the claims assigned to them. A Finance Officer sees the same for the whole department. | A-02, A-03 | M | 3 |

## 7. Business rules

| ID | Rule | Sprint |
| --- | --- | --- |
| BR-01 | **No self-approval:** no user can approve, reject, return, or make a final decision on a claim they submitted. This is enforced on the server even if the assignment data is wrong. | 1 |
| BR-02 | **Assigned reviewer only:** only a claim's assigned reviewer can approve, reject, or return it. Only a Finance Officer can make a final decision, and only on a `PendingFinance` claim. | 1 / 3 |
| BR-03 | **Valid transitions only:** a status change is accepted only if Section 5 allows it from the claim's current status. If it is rejected, the claim and its history stay unchanged. | 1 |
| BR-04 | **Reason required:** reject, return, and final reject need a non-blank comment of 5–500 characters. Approve may include an optional comment. | 1 / 2 / 3 |
| BR-05 | **Line items:** a claim has 1–20 line items. Each amount is 0.01–20,000.00 THB, with at most 2 decimal places. | 1 |
| BR-06 | **Claim total:** a claim total cannot be more than 50,000.00 THB. | 1 |
| BR-07 | **Expense date window:** an expense date cannot be in the future. It also cannot be more than 90 days before the claim's **first** submission date, so a returned claim keeps its original window. This is checked on every submission and resubmission. | 1 |
| BR-08 | **Server-authoritative totals:** totals are calculated on the server, and any total sent by the client is ignored. Line items and totals can change only while the claim is `Draft` or `Returned`. | 1 |
| BR-09 | **Two-stage approval:** a claim whose total is more than 10,000.00 THB needs a Finance Officer's final decision after the assigned reviewer approves it. Exception: if the assigned reviewer is a Finance Officer, their approval is final (ADR-0001). A claim of exactly 10,000.00 THB needs only one approval. | 3 |
| BR-10 | **Editing and deleting:** only the owner can edit a claim, and only while it is `Draft` or `Returned`. Only the owner can delete a claim, and only while it is `Draft`. | 2 |
| BR-11 | **Immutable history:** history entries can never be edited or deleted. | 1 |
| BR-12 | **Categories:** a line item's category is one of Travel, Accommodation, Meals, Supplies, Training, or Other. A Meals line item cannot be more than 1,500.00 THB. | 1 |
| BR-13 | **Concurrent decisions:** if two status changes happen on the same claim at the same time, at most one succeeds. The other gets a conflict error. | 1 |
| BR-14 | **Return limit:** a claim can be returned at most 3 times. After the third return, the reviewer can only approve or reject it. | 2 |
| BR-15 | **Assigned reviewer required:** a member without an assigned reviewer cannot create claims, including drafts. This includes every Finance Officer. | 1 |
| BR-16 | **Withdraw window:** a claim can be withdrawn only while it is `Submitted` or `Returned`. | 2 |
| BR-17 | **Reviewer fixed at submission:** a claim's assigned reviewer is recorded at its first submission and stays the same for resubmissions, even if the reporting line changes later. | 1 |

## 8. Non-functional requirements

| ID | Category | Requirement | How it is verified | Verified in |
| --- | --- | --- | --- | --- |
| NFR-01 | Security | Passwords are stored only as salted bcrypt (cost ≥ 10) or argon2 hashes. No plaintext password appears in the database, logs, or API responses. | DB inspection test, code review | Sprint 1 |
| NFR-02 | Security | Every protected endpoint returns 401 to unauthenticated requests and 403 to the wrong role or to a user without ownership/assignment. This is enforced on the server for 100% of protected endpoints. | API integration tests | Sprint 1 (re-checked every sprint) |
| NFR-03 | Security | A session expires after at most 8 hours, and logout invalidates it immediately. | Integration test | Sprint 1 |
| NFR-04 | Security | No secrets are committed to the repository. Configuration comes from environment variables, documented in `.env.example`. | Repository scan, review | Sprint 1; Lab 5 (cloud secrets) |
| NFR-05 | Reliability | Claims, line items, and history are still present after the application restarts or is redeployed. | Restart test (local); redeploy test (GCP) | Sprint 1 (local); Lab 5 (GCP) |
| NFR-06 | Integrity | A status change and its history entry are written in one database transaction. If either fails, both fail. | Integration test | Sprint 1 |
| NFR-07 | Performance | With seeded demo data (≥ 200 claims), 95% of API requests complete in under 1 second on the deployed instance. | Timed API run | Sprint 3 |
| NFR-08 | Usability | Validation errors identify the field and the rule broken (e.g. "Expense date cannot be in the future"). | UI/E2E test | Sprint 1 (claim form); Sprint 2 (edit/draft forms) |
| NFR-09 | Maintainability | Automated tests cover every BR. The Go API keeps 100% statement coverage, excluding generated code. CI runs all tests on every PR to the staging branch and to main. | `go-test-coverage` report in CI | Sprint 1 onward |
| NFR-10 | Deployability | The application builds and runs with one `docker compose up`, using only documented environment variables. | Fresh-clone build check | Lab 5 |

## 9. Role and permission summary

The detailed matrix is in `docs/system-sds.md`.

| Action | Requester | Reviewer | Finance Officer | Anonymous |
| --- | --- | --- | --- | --- |
| Log in / log out | ✓ | ✓ | ✓ | login only |
| Create, submit, or view **own** claims | ✓ | ✓ | ✗ (BR-15) | ✗ |
| Edit, withdraw, or delete own claim | allowed states | allowed states | ✗ | ✗ |
| View another user's claim | ✗ | assigned only | assigned, `PendingFinance`, and summaries | ✗ |
| Approve, reject, or return | ✗ | assigned, not own | assigned (Department Head's claims) | ✗ |
| Final approve / final reject (S3) | ✗ | ✗ | `PendingFinance` only | ✗ |
| Decision summary (S3) | ✗ | own assignments | department | ✗ |

## 10. Resolved decisions

All decisions below were made by the team on 2026-10-31.

| ID | Question | Decision | Rationale |
| --- | --- | --- | --- |
| D-01 | What kind of request? | Reimbursement of expenses already paid only. | Clear, testable rules: amounts, dates, receipts. Pre-approval plus reimbursement would be too large. |
| D-02 | Multiple roles per account? | No. Three roles: Requester, Reviewer, Finance Officer. Reviewers can also submit claims. | The spec does not require multiple roles. Reviewers submitting claims makes self-approval (BR-01) a real case to test. |
| D-03 | How are reviewers assigned? | By reporting line: Staff → Section Head → Department Head → Finance Officer, stored as each member's assigned reviewer in seed data. | Matches the brief's "assigned reviewer" and makes "unrelated reviewer" access testable. |
| D-04 | Who reviews the top of the reporting line? | The Finance Officer is the Department Head's assigned reviewer from Sprint 1. | Every requester needs a reviewer (ADR-0001). |
| D-05 | Does a Finance Officer's first-stage approval need a second stage? | No. It is final. | Finance has already reviewed the claim (ADR-0001). |
| D-06 | Can a Finance Officer submit claims? | No. They have no assigned reviewer. | Covered by BR-15 with no special case. |
| D-07 | What happens to pending claims when the reporting line changes? | They keep the reviewer recorded at submission (BR-17). | Decisions and history stay consistent. |
| D-08 | Reject vs return? | Reject is final; to try again, the requester submits a new, unrelated claim. Return lets the requester correct and resubmit the same claim to the same reviewer, at most 3 times. | Clear meaning for each decision. The limit stops endless back-and-forth. |
| D-09 | Receipt evidence? | A receipt reference number per line item. | File storage is not core to the workflow. |
| D-10 | Editing after submission? | No editing while `Submitted`. The requester may withdraw. Editing is allowed only in `Draft` or `Returned`. | A claim cannot change while a reviewer is looking at it. |
| D-11 | Can a claim be withdrawn while `PendingFinance`? | No. | The reviewer has already decided. |
| D-12 | Multi-stage approval? | Two stages for claims over 10,000.00 THB (Sprint 3). The Finance Officer may only approve or reject, not return. | A boundary rule that is easy to test. No return from stage 2 means no question about restarting stage 1. |
| D-13 | Expense date window when resubmitting? | 90 days counted from the first submission. | A slow return should not cost the requester their claim. |
| D-14 | Drafts? | Kept (Sprint 2). The owner can delete a draft. Members without an assigned reviewer cannot create drafts. | Better usability. A draft was never seen by anyone, so deleting it loses no history. |
| D-15 | Who sees reports? | Reviewers see their assigned claims; the Finance Officer sees the whole department. Requesters use filtering instead. | Avoids a separate requester report screen. |
| D-16 | Amount limits? | Claim ≤ 50,000; line item ≤ 20,000; Meals ≤ 1,500; 1–20 items. | Realistic for a department and gives clear boundary tests. |

## 11. Open questions (to confirm with instructor/TA)

| ID | Question | Proposed answer |
| --- | --- | --- |
| Q-01 | Is a seeded, fixed reporting line acceptable for "define reviewer assignment"? | Yes. The assignment rule is defined and enforced; only its administration UI is excluded (EX-08). |

## 12. Revision history

| Version | Date | Change | Author |
| --- | --- | --- | --- |
| 0.1 | 2026-10-31 | Initial baseline draft | 1005, 1045 |
| 0.2 | 2026-10-31 | Requirements review session: reporting-line assignment, Finance Officer from Sprint 1, Withdraw replaces Cancel, return limit, BR-14 to BR-17, D-01 to D-16 | 1005, 1045 |
