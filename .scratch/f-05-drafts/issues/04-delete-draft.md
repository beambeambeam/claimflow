# 04: Delete a draft

**What to build:** A requester deletes a draft they no longer need, after a confirmation. The draft and its line items are removed. Only drafts can be deleted, so a claim with history is never removed.

**Blocked by:** F-05/01 Save and view drafts

**Status:** ready-for-agent

- [ ] Policy adds `delete` (own, `DRAFT`).
- [ ] `claim.Service.Delete` removes the claim and its line items with the version check.
- [ ] `DELETE /claims/{id}` with `If-Match` → 204; non-owner → 403; `SUBMITTED` → 409 `INVALID_TRANSITION` with BR-10.
- [ ] Integration test: a claim with history cannot be deleted even through direct SQL (trigger and FK).
- [ ] Web: Delete button from `allowedActions` with a confirmation dialog; return to My Claims after delete.
