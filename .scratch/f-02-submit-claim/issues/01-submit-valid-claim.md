# 01: Submit a valid claim end to end

**What to build:** A requester opens New Claim, enters a title, a purpose, and line items (category, description, expense date, amount, receipt reference), and submits. The server calculates the total, records the requester's assigned reviewer, gives the claim a number like `CLM-2026-000123`, sets status `Submitted`, and writes the first history entry. A member with no assigned reviewer, such as a Finance Officer, gets 403.

**Blocked by:** F-01/04 Route permissions and the route inventory test. Note: the #6 contract for F-02 merges before the implementation PR.

**Status:** ready-for-agent

- [ ] Migrations add `claims`, `line_items`, and `claim_history` as in SDS §2.2, with the full status enum, CHECK constraints, indexes, and the claim number sequence.
- [ ] A `BEFORE UPDATE OR DELETE` trigger on `claim_history` raises an error; an integration test proves it (BR-11).
- [ ] A shared `money.Satang` type encodes and decodes 2-decimal strings and rejects anything else; sqlc maps `*_satang` columns to it.
- [ ] `claim.Service.Create` (submit) runs in one transaction: claim, line items, and one `SUBMIT` history row (FR-07).
- [ ] The total is computed on the server; the contract has no client total (FR-03, BR-08).
- [ ] `assigned_reviewer_id`, `first_submitted_at`, and `submitted_at` are set at submission from `user.reviewer_id` (BR-17).
- [ ] `reviewer_id` NULL → `ErrForbidden` → 403 (BR-15); `POST /claims` requires `claim:create`.
- [ ] Web: New Claim form with add/remove line-item rows, running total as `฿1,250.00`, disabled submit while sending, and a confirmation showing claim number and status.
