# 01: Decision summary screen

**What to build:** A reviewer or Finance Officer opens Summary and picks a date range. They see counts by status, approved totals by category and month, and average decision time. A reviewer sees only the claims assigned to them; a Finance Officer sees the whole department. Requesters have no Summary link and get 403 from the endpoint.

**Blocked by:** F-08/02 Finance Queue and final decisions. Note: the #6/D2 contract for F-10 merges before the implementation PR.

**Status:** ready-for-agent

- [ ] A separate report module reads the claim tables with its own queries and does not import the claim module (lint-enforced).
- [ ] `countsByStatus` counts non-draft claims in scope by current status, ignoring the date range.
- [ ] `approvedTotals` sums line items of `APPROVED` claims by Asia/Bangkok month of `decided_at` and category.
- [ ] `averageDecisionHours` = mean of `decided_at − first_submitted_at` for `APPROVED` and `REJECTED` claims decided in the range.
- [ ] Tests with a fixed dataset give exact numbers; reviewer scope excludes other reviewers; drafts never counted; month-edge boundaries.
- [ ] `GET /reports/summary?from=&to=` requires `report:read`; requester → 403.
- [ ] Web: Summary route with status tiles, month × category table, average time, empty state, date range in the URL.
