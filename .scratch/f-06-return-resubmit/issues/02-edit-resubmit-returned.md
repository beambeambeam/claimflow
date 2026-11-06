# 02: Edit and resubmit a returned claim

**What to build:** A requester edits a returned claim with the shared claim editor and resubmits it. It goes back to the same assigned reviewer's queue, history keeps every earlier decision, and the 90-day expense window still counts from the first submission.

**Blocked by:** F-06/01 Return a submitted claim; F-05/02 Edit a draft; F-05/03 Submit a draft

**Status:** ready-for-agent

- [ ] Transition table adds `RETURNED + RESUBMIT → SUBMITTED`; `submit` records `RESUBMIT` for returned claims and `SUBMIT` for drafts.
- [ ] Resubmit keeps `assigned_reviewer_id` even after the requester's `reviewer_id` changes (BR-17).
- [ ] BR-07 at resubmission uses `first_submitted_at`: expense 85 days before first submission, resubmitted 20 days later → accepted; future date → rejected.
- [ ] Edit of a `RETURNED` claim re-runs BR-05/06/12 and recalculates the total.
- [ ] Web: Edit and Submit work on returned claims.
- [ ] E2E: submit → return → edit → resubmit → approve; history shows every step.
