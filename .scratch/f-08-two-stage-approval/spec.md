# F-08: Two-stage high-value approval

Status: ready-for-agent Sprint: 3 Requirements: FR-07, FR-12, BR-01, BR-02, BR-03, BR-04, BR-09, NFR-08 Verifies: SR-01, SR-04, SR-06 (new paths) Depends on: F-04 Design: SDS §3.2 `APPROVE` → `PENDING_FINANCE`, `FINAL_APPROVE`, `FINAL_REJECT`; §4.3 `final-approve`, `final-reject`; §5.2; §5.3 `scope=finance`; §7.1 Finance Queue; ADR-0001

## Problem Statement

Large claims cost the department real money, but today a Section Head can approve a 40,000 THB claim alone and Finance only finds out later. The department wants Finance to look at every high-value claim before it is approved, without adding a pointless second step when Finance already reviewed it.

## Solution

When an assigned reviewer who is not a Finance Officer approves a claim whose total is more than 10,000.00 THB, the claim moves to `PendingFinance` instead of `Approved`. A Finance Officer sees it in a Finance Queue and makes the final decision: approve or reject with a reason, but not return. If the assigned reviewer is already a Finance Officer, their approval is final (ADR-0001). A claim of exactly 10,000.00 THB needs only one approval. From this sprint, Finance Officers can see every non-draft claim in the department.

## User Stories

1. As a department, I want claims over 10,000.00 THB to need a Finance Officer's final decision, so that Finance checks every large payout (BR-09).
2. As a reviewer, I want my approval of a high-value claim to send it to Finance, so that I know it is not yet approved (FR-12).
3. As a reviewer, I want to see clearly that the claim now waits for Finance, so that I do not think it is finished.
4. As a requester, I want a purple "Pending finance" badge, so that I know my claim is in the second stage.
5. As a requester, I want a claim of exactly 10,000.00 THB to need only one approval, so that the rule is predictable (BR-09).
6. As a Finance Officer, I want a Finance Queue of all `PendingFinance` claims, so that I see every claim waiting for a final decision.
7. As a Finance Officer, I want to open a pending claim and see line items, total, and full history including the first approval, so that I decide with all the facts.
8. As a Finance Officer, I want to final-approve a claim, so that it becomes `Approved` (FR-12).
9. As a Finance Officer, I want to final-reject a claim with a required reason of 5–500 characters, so that the requester knows why (FR-12, BR-04).
10. As a Finance Officer, I want no Return option for final decisions, so that there is no question of restarting the first stage (D-12).
11. As a Finance Officer, I want my approval of the Department Head's high-value claim to be final, so that I do not decide the same claim twice (ADR-0001, D-05).
12. As a Finance Officer, I want to be unable to make a final decision on a claim I submitted, so that self-approval is impossible in every stage (BR-01). (Finance Officers cannot create claims, but the rule holds anyway.)
13. As a security reviewer, I want reviewers and requesters blocked from final decisions with 403, so that only Finance decides the second stage (BR-02).
14. As a security reviewer, I want final decisions allowed only on `PendingFinance` claims, so that no stage is skipped (BR-02, BR-03).
15. As a requester, I want to be unable to withdraw a `PendingFinance` claim, so that a reviewer's approval is not pulled back (D-11).
16. As an auditor, I want the first approval and the final decision both in history, so that both stages are on record (FR-07).
17. As an auditor, I want two concurrent final decisions to give one success and one conflict, so that a claim never has two final decisions (BR-13).
18. As a Finance Officer, I want to see any non-draft claim in the department, so that I can answer questions about any payout (Visible Claim).
19. As a Finance Officer, I want the Finance Queue link in the header, so that I reach it in one click.
20. As a developer, I want `APPROVE` to pick its target status from the total and the approver's role in one place, so that the rule is not duplicated.
21. As a developer, I want the high-value threshold as a named constant, so that code and tests agree.

## Implementation Decisions

- **Lifecycle.** `APPROVE` picks its target: `PENDING_FINANCE` if the total is above 10,000.00 THB and the approver's role is not `finance`, otherwise `APPROVED`. Add `PENDING_FINANCE + FINAL_APPROVE → APPROVED` and `PENDING_FINANCE + FINAL_REJECT → REJECTED`. `decided_at` is set when the claim reaches `APPROVED` or `REJECTED`, not at `PENDING_FINANCE`.
- **Policy.** Add `final-approve` and `final-reject` (actor role `finance`, not own, status `PENDING_FINANCE`). Visibility: Finance sees every non-draft claim. `withdraw` stays limited to `SUBMITTED` and `RETURNED`, so `PENDING_FINANCE` cannot be withdrawn.
- **Route permission.** `finance:list` for the Finance Queue (role `finance` only).
- **Claim module.** `Act(FINAL_APPROVE)` and `Act(FINAL_REJECT)` (BR-04 reason) with the same version check and one transaction. `List` scope `finance`: status `PENDING_FINANCE`, oldest first.
- **API.** `POST /claims/{id}/final-approve` `{ comment? }`, `POST /claims/{id}/final-reject` `{ comment }`, both with `If-Match`; `GET /claims?scope=finance`.
- **Web.** Finance Queue route under `_auth` (Finance only). Final-decision panel on the detail page from `allowedActions`: approve with confirmation, reject with reason dialog, no return. "Pending finance" badge.

## Testing Decisions

- **Module against real PostgreSQL:** reviewer approves 10,000.00 → `APPROVED`; 10,000.01 → `PENDING_FINANCE`; Finance as assigned reviewer approves 40,000.00 (`rev.prasert`'s claim) → `APPROVED`; final approve and final reject with reason boundaries; reviewer or requester final decision → `ErrForbidden`; final decision on `SUBMITTED` → `ErrInvalidTransition`; withdraw `PENDING_FINANCE` → `ErrInvalidTransition{BR-16}` (D-11); Finance can `Get` any non-draft claim but not a draft; two concurrent final decisions → one `ErrConflict`; history has both stages.
- **Policy matrix:** add Finance rows for every status.
- **HTTP:** new operations covered by the route inventory; `scope=finance` 403 for reviewers.
- **Web:** Finance Queue, final-decision panel without return, badge.
- **E2E:** requester submits 15,000.00 → reviewer approves → Finance final-approves → requester sees `Approved` with two decisions in history.
- Prior art: F-04 decision and concurrency tests.

## Out of Scope

- Requiring a different Finance Officer for the final decision (rejected in ADR-0001).
- Return from the final stage (D-12).
- Payment after approval (EX-02).

## Further Notes

- This is a good candidate for the "at least one Sprint 3 workflow implemented and tested" D2 requirement.
