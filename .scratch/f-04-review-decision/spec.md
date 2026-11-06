# F-04: Review decision

Status: ready-for-agent Sprint: 1 Requirements: FR-05, FR-06, FR-07, BR-01, BR-02, BR-03, BR-04, BR-11, BR-13, NFR-06, NFR-08 Establishes: SR-04 (lifecycle integrity) Verifies: SR-01, SR-03 (local restart), SR-06 Depends on: F-01, F-02 Design: SDS §3.2 lifecycle (`APPROVE`, `REJECT`); §3.3 conditional update; §4.1 versions; §4.3 `approve`, `reject`; §5.2; §5.3 `scope=review` and required access tests; §7.1 Review Queue; §7.2 decisions and 409

## Problem Statement

Approvals happen by email today. Nobody can prove who approved what, a member can approve their own claim, and two people can decide on the same claim at once. Reviewers need one queue of the claims assigned to them and a safe way to approve or reject each one, with every decision recorded.

## Solution

A reviewer, or a Finance Officer acting as the Department Head's assigned reviewer, opens the Review Queue and sees the `Submitted` claims assigned to them. On the claim detail they approve (with an optional comment) or reject (with a required reason of 5–500 characters). The status change and its history entry are written in one transaction. The server blocks self-approval, unrelated reviewers, decisions from the wrong status, and the second of two concurrent decisions. This completes the Sprint 1 journey: submit → decide → see the result with history.

## User Stories

1. As a reviewer, I want a Review Queue of the `Submitted` claims assigned to me, so that I know what is waiting (FR-05).
2. As a reviewer, I want each queue row to show claim number, requester, title, total, and submission date, so that I can choose what to review.
3. As a reviewer, I want the oldest submitted claims first, so that nobody waits too long.
4. As a reviewer with an empty queue, I want a clear empty message, so that I know nothing is waiting.
5. As a reviewer, I want to open a claim from my queue and see its line items, total, and full history, so that I can decide with all the facts (FR-05).
6. As a reviewer, I want to approve a claim with an optional comment, so that the requester knows it is accepted (FR-06).
7. As a reviewer, I want approve to ask for confirmation, so that I do not approve by accident.
8. As a reviewer, I want to reject a claim with a required reason, so that the requester knows why (FR-06, BR-04).
9. As a reviewer, I want to be told when my reason is blank, shorter than 5, or longer than 500 characters, so that I write a useful reason (BR-04, NFR-08).
10. As a reviewer, I want the approve and reject buttons shown only when I may use them, so that I am not offered impossible actions.
11. As a reviewer, I want the claim to leave my queue after I decide, so that the queue shows only open work.
12. As a reviewer, I want to see a message and a reloaded claim when someone else changed it first, so that I never decide on stale data (BR-13).
13. As a Finance Officer, I want the Department Head's submitted claims in my Review Queue, so that the top of the reporting line is reviewed (ADR-0001).
14. As a Finance Officer, I want my approval of the Department Head's claim to be final, so that the same claim is not reviewed twice (ADR-0001).
15. As a requester, I want to see my claim as `Approved` or `Rejected` with the decision in its history, so that I know the result (FR-07).
16. As a requester, I want the reviewer's reason shown on a rejected claim, so that I know what went wrong.
17. As a reviewer who submits my own claims, I want to be unable to decide on them, even if assignment data is wrong, so that self-approval never happens (BR-01).
18. As a security reviewer, I want an unrelated reviewer blocked with 403, so that only the assigned reviewer decides (BR-02).
19. As a security reviewer, I want a requester calling a review endpoint blocked with 403, so that hiding the button is not the only protection.
20. As a security reviewer, I want requesters blocked from the Review Queue, so that they cannot list other people's claims.
21. As an auditor, I want a decision on a claim that is not `Submitted` rejected with no history written, so that ended claims stay ended (BR-03).
22. As an auditor, I want the status change and its history entry written together or not at all, so that history always matches the status (NFR-06).
23. As an auditor, I want history entries impossible to edit or delete, so that decisions cannot be rewritten (BR-11).
24. As an auditor, I want two concurrent decisions to result in exactly one success and one conflict, so that a claim never has two decisions (BR-13).
25. As a developer, I want every change to an existing claim to require `If-Match` with the claim's version, so that stale screens and races both give a clear 409.
26. As a developer, I want a missing `If-Match` to return 428, so that clients cannot skip the version check.
27. As a developer, I want every action endpoint to be one call to `claim.Service.Act`, so that all transitions go through the same checks.
28. As a developer, I want one transition table in code, so that every later action is added in one place.
29. As a member, I want a Playwright journey that proves submit → approve → requester sees Approved with history, so that the main workflow is verified end to end.
30. As a member, I want submitted and decided claims to remain after the API restarts, so that nothing is lost (NFR-05).
31. As a demo user, I want seed data with claims in `Submitted`, `Approved`, and `Rejected`, so that every Sprint 1 screen has something to show.

## Implementation Decisions

- **Lifecycle table.** One transition table: `SUBMITTED + APPROVE → APPROVED`, `SUBMITTED + REJECT → REJECTED` in Sprint 1. Before Sprint 3, `APPROVE` always goes to `APPROVED`. Terminal states have no outgoing transitions.
- **`Act(actor, id, version, action, comment)`.** In one transaction: load the claim (`ErrNotFound`); policy visible check (`ErrForbidden`); action in allowed actions, else `ErrForbidden` (BR-01, BR-02) or `ErrInvalidTransition{Rule}` (BR-03); field rules (BR-04 comment); `UPDATE … WHERE id = $id AND version = $expected` with `version + 1` and `decided_at`; zero rows → `ErrConflict`; insert history; commit.
- **Policy.** Extends F-03: `approve`, `reject` when assigned, not own (by `requester_id`), status `SUBMITTED`.
- **Scope `review`.** `List` returns claims where the actor is the assigned reviewer and status is `SUBMITTED`, oldest first. Route permission `review:list` (reviewer, finance).
- **API.** `POST /claims/{id}/approve` `{ comment? }` and `POST /claims/{id}/reject` `{ comment }`, both requiring `If-Match`. 200 returns the updated claim with new `ETag`. Errors: 403 (BR-01, BR-02, no claim data), 404, 409 `INVALID_TRANSITION` with the rule, 409 `CONFLICT`, 422 (BR-04), 428 without `If-Match`. `GET /claims?scope=review` added.
- **Web.** Review Queue route under `_auth`. A decision panel on the claim detail, shown from `allowedActions`: Approve opens a confirmation, Reject opens a dialog with a required reason field. A 409 shows the toast "This claim was changed by someone else – reloaded" and refetches. Route files combine the claims and reviews features; neither imports the other.
- **Seed.** Add sample claims in `SUBMITTED`, `APPROVED`, and `REJECTED`, each with matching history, idempotent by claim number.
- **E2E.** Playwright set up in the web app, running against the local API and web dev servers with a seeded database (Docker compose arrives in Lab 5).

## Testing Decisions

- Test through `claim.Service` against real PostgreSQL with a fixed clock, and through the full HTTP router.
- **Claim module:** approve; reject with reason lengths 4, 5, 500, 501 and blank; self-approval → `ErrForbidden` with status and history unchanged, also with corrupted assignment pointing at the requester (BR-01); unrelated reviewer → `ErrForbidden` (BR-02); decision on `APPROVED`/`REJECTED` → `ErrInvalidTransition{BR-03}` with no history row; stale version → `ErrConflict`; status and history in one transaction (force the history insert to fail and check the status did not change, NFR-06).
- **Concurrency:** two goroutines call `Act` on the same claim with the same version → exactly one success, one `ErrConflict`, one history row (BR-13).
- **HTTP access tests (SDS §5.3):** `req.anan` calls approve → 403; `rev.somchai` approves `req.anan`'s claim → 403; `rev.malee` approves her own claim → 403; `fin.wanida` approves `rev.prasert`'s claim → 200; requester `scope=review` → 403; missing `If-Match` → 428.
- **Web:** queue rows and empty state; buttons from `allowedActions` only; reason validation; 409 toast and refetch.
- **E2E:** requester submits → reviewer approves → requester sees `Approved` with history.
- **Restart check (NFR-05):** restart the API and read the same claims back (manual or scripted, recorded for #4).
- Prior art: F-02 and F-03 module and HTTP tests.

## Out of Scope

- Return and resubmit (F-06), withdraw (F-07).
- `PendingFinance` and final decisions (F-08).
- Decision summaries (F-10).

## Further Notes

- This feature closes the Theme 5 Sprint 1 example: requester submits, assigned reviewer approves or rejects with history, invalid transition rejected, self-approval prevented.
- Capture TDD red/green commits for BR-01 or BR-13 for the D1 report (#4).
