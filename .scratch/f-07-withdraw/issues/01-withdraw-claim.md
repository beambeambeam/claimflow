# 01: Withdraw a submitted or returned claim

**What to build:** A requester withdraws their own `Submitted` or `Returned` claim after a confirmation, with an optional comment. The claim becomes `Withdrawn` (grey, struck through), stays in My Claims with its history, and leaves the reviewer's queue. Withdraw is refused for any other status and for anyone but the owner.

**Blocked by:** F-06/01 Return a submitted claim. Note: the #6/D2 contract for F-07 merges before the implementation PR.

**Status:** ready-for-agent

- [ ] Transition table adds `SUBMITTED, RETURNED + WITHDRAW → WITHDRAWN`; `WITHDRAWN` is terminal.
- [ ] Policy adds `withdraw` (own, `SUBMITTED` or `RETURNED`).
- [ ] Withdraw from `APPROVED`, `REJECTED`, `WITHDRAWN` → 409 `INVALID_TRANSITION` with BR-16, no history; reviewer withdraw → 403.
- [ ] A reviewer decision after withdraw → BR-03; a stale screen → 409 `CONFLICT`.
- [ ] `POST /claims/{id}/withdraw` `{ comment? }` with `If-Match`.
- [ ] Web: Withdraw button from `allowedActions` with confirmation; Withdrawn badge.
- [ ] E2E: submit → withdraw → requester sees `Withdrawn`; the reviewer queue no longer lists it.
