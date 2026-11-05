# 01: My Claims list

**What to build:** A requester opens My Claims and sees their own claims, newest first, with claim number, title, total, status badge, and submission date. The list pages, shows skeleton rows while loading, a "No claims yet – Create claim" empty state, and an inline error with Try again. A Finance Officer is blocked from this list.

**Blocked by:** F-02/01 Submit a valid claim end to end. Note: the #6 contract for F-03 merges before the implementation PR.

**Status:** ready-for-agent

- [ ] `claim.Service.List` with scope `mine` returns only own claims, all statuses, newest first, paginated.
- [ ] `GET /claims?scope=mine&page=&pageSize=` requires `claim:list`; pageSize max 100; response `{ data, page, pageSize, total }`.
- [ ] Finance `scope=mine` → 403.
- [ ] Integration test: `req.ploy` never sees `req.anan`'s claims.
- [ ] Web: shared status badge, money, and date components (Asia/Bangkok); loading, empty, and error states per SDS §7.2.
