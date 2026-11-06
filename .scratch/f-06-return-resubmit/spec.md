# F-06: Return and resubmit

Status: ready-for-agent Sprint: 2 Requirements: FR-03, FR-07, FR-09, FR-10, BR-01, BR-02, BR-03, BR-04, BR-05, BR-06, BR-07, BR-08, BR-10, BR-12, BR-14, BR-17, NFR-08 Verifies: SR-01, SR-04, SR-06 (new paths) Depends on: F-04, F-05 Design: SDS §3.2 `RETURN`, `RESUBMIT`; §3.3; §4.3 `return`, `submit`; §5.2; §7.2 return count

## Problem Statement

A claim often has a small mistake: a wrong date or a missing receipt number. Rejecting it forces the requester to start a new, unrelated claim and loses the history. Reviewers need a way to send a claim back for correction, and requesters need to fix it and send the same claim again, without endless back-and-forth.

## Solution

The assigned reviewer can return a `Submitted` claim with a required reason. The claim becomes `Returned`, and the requester can edit it with the same editor used for drafts and resubmit it to the same reviewer. A claim can be returned at most 3 times; the reviewer sees "Returned n / 3" and after the third return can only approve or reject. The 90-day expense window still counts from the first submission, so a slow return does not cost the requester their claim.

## User Stories

1. As a reviewer, I want to return a submitted claim with a reason, so that the requester can fix it instead of starting over (FR-09).
2. As a reviewer, I want the reason required and 5–500 characters, so that the requester knows what to fix (BR-04).
3. As a reviewer, I want to see how many times the claim was already returned, as "Returned 2 / 3", so that I know how many chances are left (FR-09).
4. As a reviewer, I want the Return button hidden after the third return, so that I only approve or reject from then on (BR-14).
5. As a reviewer, I want a return after the limit rejected by the server, so that the limit holds even if the web is wrong (BR-14).
6. As a reviewer, I want a returned claim to leave my queue until it is resubmitted, so that my queue shows only claims waiting for me.
7. As a reviewer, I want the resubmitted claim back in my queue, so that I review the corrected version (FR-10, BR-17).
8. As a requester, I want to see that my claim was returned and the reviewer's reason, so that I know what to change.
9. As a requester, I want an amber "Returned" badge in My Claims, so that I see at a glance that it needs action.
10. As a requester, I want to edit a returned claim with the same form as drafts, so that fixing is familiar (FR-10).
11. As a requester, I want the total recalculated and all line-item rules checked again when I edit, so that the corrected claim is valid (FR-03, BR-05, BR-06, BR-08, BR-12).
12. As a requester, I want to resubmit the corrected claim to the same reviewer, so that the person who asked for the fix checks it (FR-10, BR-17).
13. As a requester, I want the 90-day window counted from the first submission when I resubmit, so that a slow return does not make my expenses too old (BR-07, D-13).
14. As a requester, I want a future expense date still rejected on resubmission, so that rules stay the same (BR-07).
15. As an auditor, I want every return and resubmission in history, so that earlier decisions are never lost (FR-07, FR-10).
16. As a security reviewer, I want only the assigned reviewer, not the owner, to return a claim, so that self-return and unrelated returns are impossible (BR-01, BR-02).
17. As a security reviewer, I want only the owner to edit or resubmit a returned claim, so that nobody else changes it (BR-10).
18. As an auditor, I want return allowed only from `Submitted`, so that ended claims stay ended (BR-03).
19. As a reviewer, I want a 409 message when the requester resubmitted while I was looking, so that I do not act on stale data (BR-13).
20. As a Finance Officer reviewing the Department Head, I want to return claims too, so that I have the same first-stage options as other reviewers.
21. As a developer, I want `submit` to record `RESUBMIT` for a returned claim and `SUBMIT` for a draft, so that history tells them apart.

## Implementation Decisions

- **Lifecycle.** Add `SUBMITTED + RETURN → RETURNED` with `return_count + 1` (only while `return_count < 3`), and `RETURNED + RESUBMIT → SUBMITTED`.
- **Policy.** Add `return` (as `approve`, and `return_count < 3`). `edit` and `submit` already cover `RETURNED` for the owner (F-05). The return limit is a named constant (3).
- **Claim module.** `Act(RETURN)` requires the BR-04 reason and increments `return_count`; past the limit → `ErrInvalidTransition{BR-14}`. `Act(SUBMIT)` on a `RETURNED` claim records `RESUBMIT`, keeps `assigned_reviewer_id` and `first_submitted_at`, updates `submitted_at`, and checks BR-07 against `first_submitted_at`. `Edit` on `RETURNED` uses the F-05 path.
- **API.** `POST /claims/{id}/return` `{ comment }` with `If-Match`. `POST /claims/{id}/submit` handles both draft submission and resubmission. Claim responses include `returnCount`.
- **Web.** Return button and reason dialog in the decision panel, labelled with "Returned n / 3", shown only when `return` is in `allowedActions`. The Edit Claim route and Submit button work for `RETURNED` claims. The detail page shows the latest return reason near the top.

## Testing Decisions

- Test through `claim.Service` against real PostgreSQL with a fixed clock, and through the HTTP router.
- **Module:** return with reason boundaries; return count 0→1→2→3 then fourth return → `ErrInvalidTransition{BR-14}`; approve and reject still allowed after the third return; owner cannot return own claim (BR-01); unrelated reviewer → `ErrForbidden` (BR-02); return from `APPROVED` → BR-03, no history; resubmit keeps reviewer even after the requester's `reviewer_id` changes (BR-17); BR-07 at resubmission uses `first_submitted_at` (expense 85 days before first submission, resubmitted 20 days later → accepted); edit of `RETURNED` re-runs rules; history has one row per return and resubmit.
- **HTTP:** return and submit status codes, `returnCount` in responses, 409 on stale version.
- **Web:** "Returned n / 3" label, Return hidden when not allowed, reason dialog validation, edit and resubmit flow.
- **E2E:** submit → return → edit → resubmit → approve, history shows every step.
- Prior art: F-04 decision tests, F-05 edit tests.

## Out of Scope

- Returns by a Finance Officer during a final decision; final decisions are approve or reject only (D-12, F-08).
- Notifications to the requester (EX-03).

## Further Notes

- Both Sprint 2 E2E journeys (return/resubmit and withdraw) are good D2 program evidence.
