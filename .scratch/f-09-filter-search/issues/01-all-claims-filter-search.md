# 01: All-claims view with filters and search

**What to build:** Every member opens an "All claims" view of the claims visible to them and narrows it by status, category, and date range, or searches by title. Filters live in the URL, so a view can be bookmarked. Filtering never widens visibility: a reviewer sees only assigned claims, a Finance Officer every non-draft claim, a requester only their own.

**Blocked by:** F-03/01 My Claims list; F-08/02 Finance Queue and final decisions. Note: the #6/D2 contract for F-09 merges before the implementation PR.

**Status:** ready-for-agent

- [ ] `List` scope `all` applies the visibility rule and filters in SQL; route permission is session only.
- [ ] Filters: `status` exact; `category` matches any line item; `from`/`to` inclusive Asia/Bangkok dates on `first_submitted_at` (drafts: `created_at`); `q` case-insensitive substring on title; all combine with AND.
- [ ] Invalid filter values → 400 problem.
- [ ] For each seed user, scope `all` returns exactly the claims the policy marks visible.
- [ ] Indexes support the filter queries.
- [ ] Web: filter bar bound to typed router search params; debounced search; "No claims match these filters" with Clear filters.
