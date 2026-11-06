# 01: Review Queue and approve

**What to build:** A reviewer opens the Review Queue and sees the `Submitted` claims assigned to them, oldest first. On the claim detail they approve with an optional comment after a confirmation. The claim becomes `Approved` and the decision appears in its history. A Finance Officer reviews the Department Head's claims the same way, and that approval is final (ADR-0001). Self-approval, unrelated reviewers, and requesters calling the endpoint are blocked on the server.

**Blocked by:** F-03/02 Claim Detail and the claim policy. Note: the #6 contract for F-04 merges before the implementation PR.

**Status:** ready-for-agent

- [ ] One transition table holds `SUBMITTED + APPROVE → APPROVED`; terminal states have no outgoing transitions.
- [ ] `claim.Service.Act` runs in one transaction: load, visibility, allowed action, field rules, conditional `UPDATE … WHERE version = $expected`, history insert (SDS §3.3).
- [ ] Policy adds `approve` for assigned, not own, `SUBMITTED`.
- [ ] `POST /claims/{id}/approve` requires `If-Match`; missing → 428; stale → 409 `CONFLICT`; success returns the new `ETag`.
- [ ] `GET /claims?scope=review` requires `review:list`; requester → 403.
- [ ] Access tests (SDS §5.3): `req.anan` approve → 403; `rev.somchai` approves `req.anan`'s claim → 403; `rev.malee` approves her own claim → 403, also with corrupted assignment (BR-01); `fin.wanida` approves `rev.prasert`'s claim → 200.
- [ ] Web: Review Queue route with empty state; approve button from `allowedActions` with confirmation; the route file combines the claims and reviews features without cross-imports.
