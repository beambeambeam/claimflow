# F-09: Filter and search

Status: ready-for-agent Sprint: 3 Requirements: FR-13 Verifies: SR-01, SR-06 (new paths); contributes to SR-05 (NFR-07) Depends on: F-03, F-04 Design: SDS §4.3 filters; §5.2 visibility; §5.3 `scope=all`; §7.1

## Problem Statement

As claims pile up, a member cannot find the one they need. A requester looking for last month's travel claim, or a Finance Officer checking every rejected meals claim, has to scroll through long lists. Members need to narrow the claims they are allowed to see.

## Solution

Every member gets an "All claims" view of the claims visible to them. They can filter by status, category, and date range, and search by title. Filters live in the URL, so a filtered view can be bookmarked and shared with someone who can see the same claims. Filtering never widens visibility: it only narrows the set the claim policy already allows.

## User Stories

1. As a requester, I want to filter my claims by status, so that I see only the ones still open (FR-13).
2. As a requester, I want to filter by category, so that I find all my travel claims (FR-13).
3. As a requester, I want to filter by a date range, so that I find claims from a given month (FR-13).
4. As a requester, I want to search by title, so that I find a claim by name (FR-13).
5. As a member, I want title search to ignore letter case and match part of the title, so that I do not need the exact words.
6. As a member, I want to combine filters, so that I can ask "approved meals claims in October".
7. As a member, I want the date range to be inclusive Bangkok dates on the first submission date, or the creation date for drafts, so that results match what I see on screen (AS-06).
8. As a member, I want a claim to match a category when any of its line items has that category, so that mixed claims are found.
9. As a reviewer, I want to filter the claims assigned to me, including decided ones, so that I can look up my past decisions.
10. As a Finance Officer, I want to filter every non-draft claim in the department, so that I can answer audit questions.
11. As a member, I want filters stored in the URL, so that I can bookmark and share a view.
12. As a member, I want a clear "No claims match these filters" message with a Clear filters action, so that I can recover from an empty result.
13. As a member, I want results paged, so that large result sets stay fast.
14. As a member, I want invalid filter values (a bad date, an unknown status) rejected with a clear message, so that I know what to fix.
15. As a security reviewer, I want filtering to return only claims the policy makes visible, so that a filter can never reveal another member's claim.
16. As a security reviewer, I want reviewers who are not the assigned reviewer to get nothing from filters, so that visibility stays tied to assignment.
17. As a developer, I want filters implemented in the list query, not in memory, so that performance holds with realistic data (NFR-07).

## Implementation Decisions

- **Scope `all`.** `List` with scope `all` returns every claim visible to the actor (own, assigned and not draft, and for Finance every non-draft claim), narrowed by filters. Route permission: session only.
- **Filters.** `status` exact; `category` matches if any line item has it; `from`/`to` inclusive Asia/Bangkok dates on `first_submitted_at`, or `created_at` for drafts; `q` case-insensitive substring on title. All filters are optional and combine with AND. Validation errors use the 400 problem shape.
- **Query.** Filters run in SQL with indexes that support them; the visibility rule is part of the same query, so the list and the policy agree. A shared visibility predicate keeps the SQL and the policy function in step, checked by a test.
- **API.** `GET /claims?scope=all&status=&category=&from=&to=&q=&page=&pageSize=`.
- **Web.** An "All claims" screen (or a filter bar on My Claims for requesters) with status, category, date range, and search controls bound to typed router search params; debounced search input; empty-result state with Clear filters.

## Testing Decisions

- **Module against real PostgreSQL:** each filter alone and combined; category match on any line item; date boundaries on Bangkok midnight; draft uses `created_at`; case-insensitive partial title match; for each seed user, scope `all` returns exactly the claims the policy marks visible (the list and the policy must agree).
- **HTTP:** invalid filter values → 400; pagination; route inventory covers the operation.
- **Web:** filter controls update the URL and the query; empty state; Clear filters.
- Prior art: F-03 list tests and the policy matrix.

## Out of Scope

- Full-text search across descriptions or line items.
- Saved filters per user.
- Export to CSV.

## Further Notes

- The NFR-07 performance run (F-10 ticket) exercises this endpoint with the `seed:load` dataset.
