# 04: Route permissions and the route inventory test

**What to build:** Each role gets a fixed set of route permissions (`claim: create, list`; `review: list`; `finance: list`; `report: read`), and a middleware returns 403 when the role does not grant an operation. `/auth/me` returns the permissions, and the header shows only the links they allow, so a Finance Officer sees no "My Claims" or "New Claim". A test walks every operation in the contract and checks that each one returns 401 without a session.

**Blocked by:** F-01/02 Log in and see who I am

**Status:** ready-for-agent

- [ ] Permissions are defined once as a statement and a per-role subset (SDS §5.1); code checks permissions, never role names.
- [ ] `RequirePermission(resource, action)` returns 403 in the problem format when the role lacks the permission.
- [ ] `/auth/me` returns each role's permission map; tested for requester, reviewer, and finance.
- [ ] The route inventory test reads every operation from the contract and asserts 401 without a session, except `/auth/login`, `/health`, and `/ready` (NFR-02).
- [ ] Web: header links come from `permissions`; Finance sees no claim links (BR-15).
