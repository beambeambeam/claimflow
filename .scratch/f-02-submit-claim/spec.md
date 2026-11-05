# F-02: Submit claim

Status: ready-for-agent Sprint: 1 Requirements: FR-02, FR-03, FR-07, BR-05, BR-06, BR-07, BR-08, BR-12, BR-15, BR-17, NFR-08 Depends on: F-01 Design: SDS §2.2 `claims`, `line_items`, `claim_history`; §2.3 money; §2.4 time; §3 claim module; §4.3 `POST /claims`; §5.2; §7.1 New Claim; §7.2

## Problem Statement

Members pay for work expenses themselves and then wait to be paid back. Today they fill in paper forms: totals are added by hand, the rules for amounts and dates are applied unevenly, and nobody knows which reviewer has the claim. A requester needs one place to submit a claim that the system checks and sends to the right reviewer.

## Solution

A requester fills in a New Claim form with a title, a purpose, and 1–20 line items (category, description, expense date, amount, receipt reference) and submits it. The server calculates the total, checks every rule, and shows a message under the exact field that breaks a rule. On success the claim gets a claim number, status `Submitted`, the requester's assigned reviewer recorded on it, and its first history entry. A member with no assigned reviewer, such as a Finance Officer, cannot create claims.

## User Stories

1. As a requester, I want to open a New Claim form, so that I can ask to be paid back.
2. As a requester, I want to enter a title and a purpose, so that my reviewer knows what the claim is for.
3. As a requester, I want to add line items one by one, so that each paid expense is listed with its receipt.
4. As a requester, I want to remove a line item before I submit, so that I can fix a mistake.
5. As a requester, I want to choose a category from Travel, Accommodation, Meals, Supplies, Training, or Other, so that expenses are classified the same way for everyone (BR-12).
6. As a requester, I want to enter a receipt reference number per line item, so that the reviewer can match each line item to a receipt (D-09).
7. As a requester, I want to see a running total while I type, so that I know roughly what I am claiming.
8. As a requester, I want the server to calculate the real total, so that the amount I am paid is always correct (FR-03, BR-08).
9. As a requester, I want to be told when I have no line items or more than 20, so that I submit a valid claim (BR-05).
10. As a requester, I want to be told when an amount is 0, more than 20,000.00, or has more than 2 decimals, so that I fix it before submitting (BR-05).
11. As a requester, I want to be told when my total is more than 50,000.00, so that I split large expenses correctly (BR-06).
12. As a requester, I want to be told when a Meals line item is more than 1,500.00, so that I follow the meals limit (BR-12).
13. As a requester, I want to be told when an expense date is in the future, so that I only claim expenses I have already paid (BR-07).
14. As a requester, I want to be told when an expense date is more than 90 days before submission, so that I know the claim window (BR-07).
15. As a requester, I want "today" to mean today in Bangkok, so that a claim submitted just after midnight is checked correctly (AS-06).
16. As a requester, I want each error shown under the field that broke the rule, with the rule in plain words, so that I can fix it fast (NFR-08).
17. As a requester, I want the form to check simple limits before sending, so that I see obvious mistakes at once.
18. As a requester, I want my input kept after a server error, so that I do not type everything again.
19. As a requester, I want the Submit button disabled while the request runs, so that I do not create the claim twice.
20. As a requester, I want my claim sent to my assigned reviewer when I submit it, so that I do not need to know who reviews me (FR-02).
21. As a requester, I want a claim number like `CLM-2026-000123`, so that I can refer to my claim.
22. As a requester, I want to see my new claim with status `Submitted` after I submit, so that I know it worked.
23. As a reviewer, I want to submit my own claims, so that I am paid back like any other member; they go to my own assigned reviewer.
24. As a Finance Officer, I want to be told I cannot create claims, so that I understand why the action is not there (BR-15).
25. As a member without an assigned reviewer, I want creation blocked on the server, so that no claim exists with nobody to decide on it (BR-15).
26. As an assigned reviewer, I want the reviewer recorded on the claim at first submission, so that a later reporting-line change does not move the claim away from me (BR-17).
27. As an auditor, I want every submission to write a history entry with previous status, new status, actor, time, and comment, so that I can see what happened (FR-07).
28. As an auditor, I want history entries to be impossible to change or delete, so that the record is trustworthy (BR-11).
29. As a security reviewer, I want any total sent by the client ignored, so that nobody can change the amount by editing the request (BR-08).
30. As a security reviewer, I want the requester taken from the session, not the request, so that nobody can submit a claim as someone else.
31. As a developer, I want money handled as integer satang and sent as 2-decimal strings, so that there are no rounding errors (SDS §2.3).
32. As a developer, I want the limits defined once as named constants, so that the code and tests use the same values.
33. As a developer, I want database CHECK constraints on amounts, totals, and the reviewer, so that a bug in the code cannot store an invalid claim.

## Implementation Decisions

- **Claim module.** One `claim.Service` holds all claim rules. This feature implements `Create(actor, input, submit=true)`. It loads the actor's `reviewer_id`, rejects with `ErrForbidden` (BR-15) when it is NULL, runs field rules, computes the total, and in one transaction inserts the claim, its line items, and the first history row (`SUBMIT`, from NULL to `SUBMITTED`).
- **Rules.** Pure functions with named constants: item 0.01–20,000.00 (BR-05), 1–20 items (BR-05), total ≤ 50,000.00 (BR-06), Meals ≤ 1,500.00 (BR-12), expense date ≤ today and ≥ first-submission date − 90 days in Asia/Bangkok (BR-07). They return a `ValidationError` with one entry per field (`items[1].expenseDate`), the rule ID, and a user message.
- **Money.** A shared `money.Satang` type (`int64`). JSON is a decimal string with exactly 2 places; decoding rejects anything else, including `"1.005"`. The contract pattern is `^\d{1,6}\.\d{2}$`. A sqlc override maps `*_satang` columns to it.
- **Time.** The claim module takes the current time from an injected clock. "Today" is the Asia/Bangkok calendar date.
- **Schema.** `claims` (claim number from a never-resetting sequence `CLM-<Bangkok year>-<6 digits>`, requester, assigned reviewer with CHECK not equal to requester and CHECK not NULL unless `DRAFT`, title, purpose, status enum with all seven statuses, `total_satang` CHECK, `return_count`, `version`, `first_submitted_at`, `submitted_at`, `decided_at`), `line_items` (line number 1–20 unique per claim, category enum, CHECK amount and Meals cap), and `claim_history` (identity PK for order, action enum, from/to status, comment ≤ 500). A `BEFORE UPDATE OR DELETE` trigger on `claim_history` raises an error (BR-11). Indexes on `(requester_id, status)` and `(assigned_reviewer_id, status)`.
- **Assigned reviewer.** Copied from `user.reviewer_id` at first submission, with `first_submitted_at` and `submitted_at` set to now. Never changed later (BR-17).
- **API.** `POST /claims` with `{ title, purpose, items[] }`, route permission `claim:create`. Returns 201 with `{ data: Claim }`, an `ETag` with the version. Errors: 400 for contract shape, 403 for BR-15, 422 for business rules. Any client `total` field is not in the contract and is ignored.
- **Web.** New Claim screen with TanStack Form and the generated Zod schemas for quick client checks; line-item rows that can be added and removed; a running total shown as `฿1,250.00`; server problem `errors` mapped to fields; disabled submit with spinner. After success, go to the claim's detail page (F-03 adds the full detail; until then a minimal confirmation with claim number and status is fine).

## Testing Decisions

- Test external behaviour through `claim.Service` against real PostgreSQL with a fixed clock; do not test private helpers directly.
- **Boundaries (table-driven):** 0 and 21 items; amounts `0.00`, `0.01`, `20000.00`, `20000.01`, `1.005`; total 50,000.00 vs 50,000.01; Meals 1,500.00 vs 1,500.01; expense date tomorrow, today, 90 and 91 days ago, and just after Bangkok midnight; invalid category.
- **Behaviour:** client-sent total ignored (BR-08); Finance Officer creation → `ErrForbidden` (BR-15); assigned reviewer recorded and equal to the requester's `reviewer_id` (BR-17); exactly one history row written; history row cannot be updated or deleted (BR-11 trigger test).
- **Value type tests** for `money.Satang` JSON encoding and decoding.
- **HTTP:** `POST /claims` status codes and problem bodies, `claim:create` 403 for Finance, field paths in `errors`.
- **Web:** form validation messages per field (NFR-08), add/remove rows, server errors mapped to fields, double-submit prevention.
- Prior art: the auth module tests and HTTP tests from F-01.

## Out of Scope

- Saving as draft, editing, deleting (F-05).
- Viewing lists and claim detail with history (F-03).
- Two-stage approval for high-value claims (F-08).
- Receipt file upload and OCR (EX-05); multiple currencies (EX-06).

## Further Notes

- F-05 and F-06 reuse the rules and the claim form. Keep the form a reusable component that takes initial values, so editing can use it later.
- The claim status enum includes Sprint 2 and 3 statuses from the start, so later features need no enum migration.
