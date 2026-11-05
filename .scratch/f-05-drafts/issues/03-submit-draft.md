# 03: Submit a draft

**What to build:** A requester submits a finished draft from its detail page. This is its first submission: the assigned reviewer is recorded, the 90-day expense window is checked from now, the claim becomes `Submitted`, and the first history entry is written (`Draft → Submitted`). It then appears in the reviewer's queue.

**Blocked by:** F-05/01 Save and view drafts; F-04/01 Review Queue and approve

**Status:** ready-for-agent

- [ ] Transition table adds `DRAFT + SUBMIT → SUBMITTED`; policy adds `submit` (own, `DRAFT`, owner has an assigned reviewer).
- [ ] `Act(SUBMIT)` sets `assigned_reviewer_id`, `first_submitted_at`, `submitted_at` (BR-17) and checks BR-07 with the fixed clock.
- [ ] Exactly one history row with `from_status = DRAFT` (FR-07).
- [ ] `POST /claims/{id}/submit` with `If-Match`.
- [ ] Web: Submit button on the draft detail from `allowedActions`.
