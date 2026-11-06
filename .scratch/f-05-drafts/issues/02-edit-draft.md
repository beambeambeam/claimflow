# 02: Edit a draft

**What to build:** A requester opens Edit on their draft and changes the title, purpose, and line items in the same form used for New Claim. Every save re-runs the line-item rules and recalculates the total on the server. A save from a stale tab gets a 409 message instead of overwriting newer work. The edit path also accepts `Returned` claims, for F-06.

**Blocked by:** F-05/01 Save and view drafts

**Status:** ready-for-agent

- [ ] Policy adds `edit` (own, `DRAFT` or `RETURNED`).
- [ ] `claim.Service.Edit` replaces title, purpose, and line items, recalculates the total, runs BR-05/06/12, and uses the version check.
- [ ] `PUT /claims/{id}` with `If-Match`; non-owner → 403; `SUBMITTED` → 409 `INVALID_TRANSITION` with BR-10; stale → 409 `CONFLICT`.
- [ ] A client total is ignored (BR-08).
- [ ] Web: Edit Claim route reusing the claim form with initial values; Edit button from `allowedActions`; field messages (NFR-08).
