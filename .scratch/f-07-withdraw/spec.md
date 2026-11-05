# F-07: Withdraw

Status: ready-for-agent Sprint: 2 Requirements: FR-07, FR-11, BR-03, BR-16 Verifies: SR-01, SR-04, SR-06 (new paths) Depends on: F-02, F-06 Design: SDS §3.2 `WITHDRAW`; §4.3 `withdraw`; §5.2

## Problem Statement

Sometimes a requester submits a claim by mistake, or a trip is refunded by someone else. They cannot edit a submitted claim (D-10), and asking the reviewer to reject it puts a rejection on the record for no reason. The requester needs to take their own claim back while it is still open.

## Solution

The requester can withdraw their own claim while it is `Submitted` or `Returned`. Withdrawing ends the claim with status `Withdrawn` and writes a history entry. A claim that has been decided, or that waits for a Finance Officer's final decision, cannot be withdrawn.

## User Stories

1. As a requester, I want to withdraw my submitted claim, so that a mistaken claim does not get reviewed (FR-11).
2. As a requester, I want to withdraw my returned claim instead of fixing it, so that I can drop a claim I no longer need (FR-11).
3. As a requester, I want a confirmation before withdrawing, so that I do not end my claim by accident.
4. As a requester, I want an optional comment when I withdraw, so that my reviewer knows why.
5. As a requester, I want the Withdraw button only when withdrawing is allowed, so that I am not offered an action that will fail.
6. As a requester, I want a struck-through grey "Withdrawn" badge, so that I see the claim is closed.
7. As a requester, I want a withdrawn claim to stay in My Claims with its history, so that I keep a record.
8. As a reviewer, I want a withdrawn claim removed from my queue, so that I do not waste time on it.
9. As a reviewer, I want a 409 message if the requester withdrew while I was deciding, so that I do not decide on an ended claim (BR-13).
10. As an auditor, I want a withdraw recorded in history with actor and time, so that I can see who ended the claim (FR-07).
11. As an auditor, I want withdraw rejected for `Approved`, `Rejected`, and `Withdrawn` claims, so that ended claims stay ended (BR-03, BR-16).
12. As an auditor, I want withdraw rejected for `PendingFinance` claims, so that a claim the reviewer already approved cannot be pulled back (D-11).
13. As a security reviewer, I want only the owner to withdraw, so that a reviewer cannot withdraw a claim to avoid deciding it.
14. As a developer, I want withdraw to be one more row in the transition table and policy, so that it follows the same checks as every action.

## Implementation Decisions

- **Lifecycle.** Add `SUBMITTED + WITHDRAW → WITHDRAWN` and `RETURNED + WITHDRAW → WITHDRAWN`. `WITHDRAWN` is terminal.
- **Policy.** Add `withdraw` (own, status `SUBMITTED` or `RETURNED`).
- **Claim module.** `Act(WITHDRAW)` with an optional comment; any other status → `ErrInvalidTransition{BR-16}`; non-owner → `ErrForbidden`.
- **API.** `POST /claims/{id}/withdraw` `{ comment? }` with `If-Match`.
- **Web.** Withdraw button on the detail page from `allowedActions`, with a confirmation dialog and optional comment.

## Testing Decisions

- **Module against real PostgreSQL:** withdraw from `SUBMITTED` and from `RETURNED` → `WITHDRAWN` with one history row; withdraw from `APPROVED`, `REJECTED`, `WITHDRAWN` → `ErrInvalidTransition{BR-16}` with no history; assigned reviewer withdraw → `ErrForbidden`; stale version → `ErrConflict`; reviewer decision after withdraw → `ErrInvalidTransition{BR-03}`.
- **HTTP:** status codes and 428 without `If-Match`.
- **Web:** button from `allowedActions`, confirmation, badge.
- **E2E:** submit → withdraw → requester sees `Withdrawn`; reviewer queue no longer lists it.
- The `PendingFinance` case (D-11) is tested in F-08 once that status exists.
- Prior art: F-04 and F-06 action tests.

## Out of Scope

- Undoing a withdraw. A withdrawn claim is ended; the requester submits a new claim.
- Deleting submitted claims (only drafts can be deleted, BR-10).

## Further Notes

- Depends on F-06 only because withdrawing from `Returned` needs that state to exist.
