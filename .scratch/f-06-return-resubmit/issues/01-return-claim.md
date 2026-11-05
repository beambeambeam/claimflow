# 01: Return a submitted claim

**What to build:** The assigned reviewer returns a `Submitted` claim with a required reason. The claim becomes `Returned` with an amber badge, the requester sees the reason, and it leaves the reviewer's queue. The Return button shows "Returned n / 3"; after the third return it is gone and the server refuses another return.

**Blocked by:** F-04/02 Reject, invalid transitions, and concurrent decisions. Note: the #6/D2 contract for F-06 merges before the implementation PR.

**Status:** ready-for-agent

- [ ] Transition table adds `SUBMITTED + RETURN → RETURNED` with `return_count + 1`; the limit 3 is a named constant.
- [ ] Policy adds `return` (as `approve`, and `return_count < 3`).
- [ ] Reason rule BR-04 applies; fourth return → 409 `INVALID_TRANSITION` with BR-14; approve and reject still allowed after the third.
- [ ] Owner returning own claim → 403 (BR-01); unrelated reviewer → 403 (BR-02); return from `APPROVED` → BR-03 with no history.
- [ ] `POST /claims/{id}/return` with `If-Match`; responses include `returnCount`.
- [ ] Web: Return button with "Returned n / 3" and reason dialog, shown only from `allowedActions`; latest return reason shown on the detail page.
