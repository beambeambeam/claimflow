# 01: High-value approval goes to Pending finance

**What to build:** When an assigned reviewer who is not a Finance Officer approves a claim whose total is more than 10,000.00 THB, the claim becomes `PendingFinance` (purple badge) instead of `Approved`. A claim of exactly 10,000.00 THB, or one approved by a Finance Officer as assigned reviewer, is approved at once (ADR-0001). A `PendingFinance` claim cannot be withdrawn.

**Blocked by:** F-04/02 Reject, invalid transitions, and concurrent decisions; F-07/01 Withdraw a submitted or returned claim. Note: the #6/D2 contract for F-08 merges before the implementation PR.

**Status:** ready-for-agent

- [ ] `APPROVE` picks its target in one place: `PENDING_FINANCE` if total > 10,000.00 (named constant) and approver role is not `finance`, otherwise `APPROVED`.
- [ ] `decided_at` stays NULL at `PENDING_FINANCE`.
- [ ] Tests: 10,000.00 → `APPROVED`; 10,000.01 → `PENDING_FINANCE`; `fin.wanida` approves `rev.prasert`'s 40,000.00 claim → `APPROVED`.
- [ ] Withdraw of a `PENDING_FINANCE` claim → 409 `INVALID_TRANSITION` with BR-16 (D-11).
- [ ] Web: "Pending finance" badge; the reviewer sees that the claim waits for Finance.
