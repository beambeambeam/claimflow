# F-10: Decision summaries

Status: ready-for-agent Sprint: 3 Requirements: FR-14 Establishes: SR-05 (performance, NFR-07) Verifies: SR-01, SR-06 (new paths) Depends on: F-04, F-08 Design: SDS §4.4 decision summary; §5.3 matrix; §7.1 Summary; §2.5 `seed:load`

## Problem Statement

Reviewers and Finance have no overview. A Section Head cannot see how many claims wait for them or how long their decisions take, and Finance cannot see what the department spent by category each month. They also need to know the system stays fast with a realistic amount of data.

## Solution

Reviewers and Finance Officers get a Summary screen. It shows counts by status, the approved total by category and month, and the average decision time for a chosen date range. A reviewer sees only the claims assigned to them; a Finance Officer sees the whole department. Requesters use filtering instead (D-15). A load dataset of at least 200 claims and a timed API run prove that 95% of requests finish in under 1 second on the deployed instance (NFR-07).

## User Stories

1. As a reviewer, I want counts of my assigned claims by status, so that I see my workload (FR-14).
2. As a reviewer, I want the approved total by category and month for my assigned claims, so that I see what I approved (FR-14).
3. As a reviewer, I want my average decision time, so that I know how quickly I respond (FR-14).
4. As a Finance Officer, I want the same summary for the whole department, so that I can report on spending (FR-14).
5. As a reviewer or Finance Officer, I want to choose a date range, so that I can look at one month or a quarter.
6. As a reviewer or Finance Officer, I want counts by status to show every current claim regardless of the date range, so that the workload view is always current.
7. As a reviewer or Finance Officer, I want approved totals and average decision time to use the decision date, so that a month's numbers mean "decided in that month".
8. As a reviewer or Finance Officer, I want money shown as `฿18,250.00` and months as `Oct 2026`, so that the numbers are easy to read.
9. As a reviewer or Finance Officer, I want a clear empty state when there are no decisions in the range, so that zero is not confused with an error.
10. As a requester, I want no Summary link, so that I am not shown a screen I cannot use (D-15).
11. As a security reviewer, I want requesters blocked from the summary endpoint with 403, so that hiding the link is not the only protection.
12. As a security reviewer, I want a reviewer's summary to include only claims assigned to them, so that they learn nothing about other sections.
13. As a developer, I want the summary computed in SQL from the claim tables, so that it stays fast and does not depend on the claim module internals.
14. As an operator, I want a command that loads at least 200 extra synthetic claims, so that performance can be measured with realistic data (NFR-07).
15. As an operator, I want a repeatable timed run of the main API requests on the deployed instance, so that I can prove 95% finish in under 1 second (NFR-07).
16. As an operator, I want the timing results recorded with the tested version, so that the D2 and final reports can cite them.

## Implementation Decisions

- **Report module.** A separate feature from the claim module; it reads the claim tables with its own queries and must not import the claim module (lint-enforced boundary). It owns `Summary(actor, from, to)`.
- **Scope.** Reviewer: claims where they are the assigned reviewer. Finance: every non-draft claim.
- **Fields.** `countsByStatus` counts non-draft claims in scope by current status, ignoring the date range. `approvedTotals` sums line-item amounts of `APPROVED` claims grouped by Asia/Bangkok month of `decided_at` and by category. `averageDecisionHours` is the mean of `decided_at − first_submitted_at` for `APPROVED` and `REJECTED` claims decided in the range.
- **API.** `GET /reports/summary?from=&to=`, route permission `report:read` (reviewer, finance). Response shape as SDS §4.4.
- **Web.** Summary route under `_auth` for roles with `report:read`: status count tiles, a month × category table (or chart) of approved totals, average decision time, and a date-range picker in the URL.
- **Load data.** A `seed:load` command adds ≥ 200 synthetic claims across statuses, categories, and months, with history, idempotent by claim number. Indexes are checked against the summary, list, and filter queries.
- **Timed run.** A scripted run of the main requests (login, `/auth/me`, list scopes, filters, detail, an action, summary) against the deployed instance with the load data, reporting p95 per request and overall.

## Testing Decisions

- **Report module against real PostgreSQL:** a fixed dataset with known decisions gives exact counts, totals per month × category, and average hours; reviewer scope excludes other reviewers' claims; Finance scope includes all non-draft claims; date range boundaries on Bangkok month edges; drafts never counted.
- **HTTP:** requester → 403; shape matches the contract; route inventory covers it.
- **Web:** tiles, table, empty state, date-range in URL; Summary link only with `report:read`.
- **Performance (NFR-07):** timed run on the deployed instance with `seed:load` data; p95 < 1 s recorded with the release tag. This depends on the GCP deployment from Lab 5.
- Prior art: F-09 list query tests.

## Out of Scope

- Requester reports (D-15).
- Export or scheduled reports.
- Budget tracking per project or cost centre (EX-07).

## Further Notes

- The NFR-07 run belongs to SR-05 rather than to F-10 alone; it is placed here because F-10 is the last feature and the run needs every list and summary endpoint in place.
