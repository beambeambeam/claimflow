# 02: Claim Detail and the claim policy

**What to build:** A member opens a claim and sees its line items, server total, status, assigned reviewer, and history timeline. One pure policy function decides who may see a claim and which actions they may take; the detail response carries `allowedActions` so the web never re-derives rules. A member who may not see the claim gets "You don't have access to this claim" with no claim data; an unknown ID gets "Claim not found".

**Blocked by:** F-02/01 Submit a valid claim end to end

**Status:** ready-for-agent

- [ ] `Allowed(actor, claim) → (visible, actions)` is a pure function: visible if own, or assigned and not `DRAFT`; the "not own" check uses `requester_id`.
- [ ] Table-driven policy matrix tests: role × own/assigned/unrelated × status, including assignment corrupted to point at the requester.
- [ ] `claim.Service.Get` returns `ErrNotFound` for a missing claim and `ErrForbidden` for a claim that is not visible.
- [ ] `GET /claims/{id}` returns the claim with line items, history, `version`, `allowedActions`, and `ETag: "<version>"`.
- [ ] `req.ploy` reads `req.anan`'s claim → 403 with no claim fields; an unrelated reviewer → 403.
- [ ] Web: Claim Detail route with history timeline; inline 403 and 404 pages.
