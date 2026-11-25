# F-10 Decision Summaries – Engineering Contract

- **Feature:** F-10 Decision summaries (`docs/features.md`)
- **Sprint:** 3 · **Issues:** #49 (F-10/01), #50 (F-10/02)
- **Requirements:** FR-14, NFR-02, NFR-07, NFR-08
- **Responsibilities verified here:** SR-01 (new endpoint), SR-05 (performance, NFR-07)
- **Depends on:** F-08 (`PENDING_FINANCE`, final decisions, Finance visibility), F-09 (filter queries for the timed run)
- **Design baseline:** `docs/system-sds.md` v0.18 §1 `internal/report`; §2.5 `seed:load`; §4.3 `GET /reports/summary`; §4.4; §5.1 `report:read`; §5.3; §7.1 Summary
- **Version:** 0.2 · **Last updated:** 2026-11-25

---

## 1. Feature SDS

### 1.1 Behavior

- A reviewer or Finance Officer opens **Summary** and may pick a date range. The page shows three things:
  - Counts by status.
  - Approved totals by month and category.
  - The average decision time.
- A reviewer's summary covers only the claims assigned to them. A Finance Officer's summary covers every claim that is not `DRAFT` in the department (D-15).
- Requesters have no Summary link, and the endpoint returns 403 to them. They use filtering instead (F-09).
- Counts by status always show the current workload: the date range does not apply to them. Approved totals and the average decision time cover claims whose decision falls in the range.
- F-10/02 adds a load dataset of at least 200 claims and a timed run that shows 95% of API requests finish in under 1 second on the deployed instance (NFR-07).

### 1.2 Workflow

```mermaid
sequenceDiagram
    participant U as Reviewer or Finance
    participant W as Web (Summary)
    participant H as Handler
    participant R as report.Service.Summary
    participant D as PostgreSQL
    U->>W: open /reports?from=2026-10-01&to=2026-10-31
    W->>H: GET /reports/summary?from=2026-10-01&to=2026-10-31
    H->>H: session; report:read (403); dates (400)
    H->>R: Summary(actor, from, to)
    R->>D: counts by status (scope, no date range)
    R->>D: approved totals by Bangkok month × category (scope, decided in range)
    R->>D: average decision hours (scope, decided in range)
    H-->>W: 200 { data: Summary }
```

### 1.3 Scope

| Actor role | Claims in scope |
| --- | --- |
| `reviewer` | `assigned_reviewer_id = actor` (never `DRAFT`, because drafts have no assigned reviewer). The reviewer's own claims are not included. |
| `finance` | Every claim whose status is not `DRAFT`. |
| `requester` | None: 403 from the route permission `report:read`. |

### 1.4 Fields

The response shape is SDS §4.4.

| Field | Rule |
| --- | --- |
| `countsByStatus` | The number of claims in scope by current status. It always has the six keys `SUBMITTED`, `RETURNED`, `PENDING_FINANCE`, `APPROVED`, `REJECTED`, `WITHDRAWN`, with `0` for an empty status. The date range does not apply. |
| `approvedTotals` | For `APPROVED` claims in scope whose `decided_at` is in the range: the sum of line-item amounts grouped by Asia/Bangkok month of `decided_at` (`YYYY-MM`) and by line-item category. A claim with two categories adds to two entries. Only non-zero groups are listed, ordered by month, then by category in `ExpenseCategory` order. `total` is a money string (`"18250.00"`). |
| `averageDecisionHours` | The mean of `decided_at − first_submitted_at`, in hours, for `APPROVED` and `REJECTED` claims in scope whose `decided_at` is in the range. It is rounded to one decimal place, half away from zero. It is `null` when no claim was decided in the range. |

For a two-stage claim, `decided_at` is the time of the final decision (F-08 §1.3). Its decision time therefore includes the Finance stage, and the claim counts in the assigned reviewer's summary as well as in Finance's.

### 1.5 Date range

- `from` and `to` are optional dates `YYYY-MM-DD`, inclusive, compared against the Asia/Bangkok date of `decided_at`.
- A missing `from` means no lower bound. A missing `to` means no upper bound. With neither, the range is all time.
- An invalid date, or `from` later than `to`, is a 400 with the F-09 §1.3 messages.

### 1.6 Failure and boundary cases

| Case | Result |
| --- | --- |
| `rev.malee` with no range | Her assigned claims only; `rev.somchai`'s and her own claims are not counted |
| `fin.wanida` with no range | Every non-draft claim |
| `req.anan` | 403 |
| No session | 401 |
| Claim decided at `2026-10-31T23:59:59+07:00` / `2026-11-01T00:00:00+07:00`, range `2026-10-01` to `2026-10-31` | In / out of October totals and the average |
| Claim decided at `2026-10-31T17:30:00Z` (1 Nov 00:30 in Bangkok) | Month `2026-11` |
| `PENDING_FINANCE`, `SUBMITTED`, `RETURNED`, `WITHDRAWN` claims | In `countsByStatus` only |
| A range with no decisions | `approvedTotals` `[]`, `averageDecisionHours` `null`, `countsByStatus` unchanged |
| `from=2026-11-10&to=2026-11-01`, `from=Oct` | 400 |

### 1.7 Module boundary

`internal/report` is its own feature package (SDS §1). It reads `claims` and `line_items` with its own sqlc queries and does not import `internal/claim`. A `depguard` rule in `apps/api/.golangci.yml` makes that import a lint error. The report module gets the actor's ID and role from `auth.UserFrom`, as the claim handler does.

### 1.8 Load dataset (F-10/02)

- `bun run seed:load` in `apps/api` runs a `seed-load` subcommand of the one binary, so it runs against the deployed database in the same way as `migrate` and `seed`. It needs the base seed users.
- It adds 240 claims with claim numbers `CLM-LOAD-0001` to `CLM-LOAD-0240`. The claim-number sequence never produces that range. Running it again adds nothing (idempotent by `claim_no`).
- The claims spread over the seed requesters and reviewers, every non-draft status, every category, and decision dates in each month from July to November 2026.
- Each claim has 1–5 line items within the BR rules, and a history that the lifecycle allows (for example, `SUBMIT`, `APPROVE`, `FINAL_APPROVE` for a two-stage claim over 10,000.00).
- At least 60 claims are `SUBMITTED` to `rev.malee`, so three timed runs of 20 rounds have claims to approve.
- **Index check:** `EXPLAIN ANALYZE` of the `mine`, `review`, `finance`, and `all` list queries (with and without filters) and of the three summary queries, on the load data. The PR records the plans. If a query on the load data misses NFR-07, an index migration is added in the same PR.

### 1.9 Timed run (F-10/02, NFR-07)

- The script `scripts/nfr07-timed-run.ts` (Bun) runs at the root with `bun run perf -- --base-url <url> --rounds 20`.
- Seed user passwords come from `SEED_USER_PASSWORD` in the environment. None are in Git (NFR-04).
- Before timing, the script counts the `SUBMITTED` load claims assigned to `rev.malee`. If there are fewer than `--rounds`, it exits non-zero with "Not enough submitted load claims to approve; reset the database and run seed:load again." It never skips the approve, so every saved run covers the same requests.
- It first sends one warm-up round of every request type except `approve`. The warm-up is not timed and not counted, so a Cloud Run cold start does not decide NFR-07.
- Each timed round sends these requests in order, each timed from send to the last body byte:
  - log in as `req.anan`, `rev.malee`, and `fin.wanida`;
  - `GET /auth/me`;
  - `scope=mine`, `review`, `finance`, and `all`;
  - `scope=all` with a status, category, date range, and `q`;
  - `GET /claims/{id}`;
  - one `approve` of a load claim;
  - `GET /reports/summary` with and without a range.
- It prints a Markdown table with the count, p50, p95, and max per request type and overall, plus the base URL, the time, and the API version from `/health`.
- It exits non-zero when the overall p95 is 1 s or more.
- The run on the deployed instance, with the load data, is saved as `docs/perf/nfr-07-<tag>.md` and cited in `docs/traceability.md` with the tag and commit.

## 2. UI spec

### 2.1 Summary (`/reports`, `_auth/reports.tsx`)

The page lives in `features/reports`.

| Element | Behavior |
| --- | --- |
| Header link | "Summary" when `report:read` is in `permissions` (F-01). |
| Date range | **From** and **To** date inputs bound to the URL search params `from` and `to`. The page opens with no range (all time). **Clear** removes both. The web checks `from` ≤ `to` before it sends. |
| Status tiles | Six tiles in lifecycle order, each with the status badge label and its count. A caption reads "Current claims, not limited by the date range." |
| Approved totals | A table with one row per month (`Oct 2026`) and one column per category with any approved total in the range. Each cell shows `฿18,250.00`, or "–" when that group has none. The last column is the month total. |
| Average decision time | "Average decision time: 26.4 hours", or "–" when `averageDecisionHours` is `null`. |

### 2.2 States

| State | Behavior |
| --- | --- |
| Loading | Skeleton tiles and table rows. |
| No decisions in range | Tiles as usual; in place of the table, "No decisions in this date range."; the average shows "–". |
| 403 | "You don't have access to this page." (a requester who opens `/reports` directly). |
| Error | Inline error with **Try again**, as F-03. |

## 3. API spec

| Method | Path | Request | Success | Errors | Auth |
| --- | --- | --- | --- | --- | --- |
| GET | `/reports/summary?from=&to=` | Optional `from`, `to` (`format: date`) | 200 `{ "data": Summary }` (SDS §4.4) | 400, 401, 403 | session; `report:read` |

Operation ID `getDecisionSummary`, tag `reports`. New schemas:

- `Summary`, with all three fields required. `averageDecisionHours` is `nullable`.
- `StatusCounts`, an object with the six required integer keys.
- `ApprovedTotal` (`month` with pattern `^\d{4}-\d{2}$`, `category` `ExpenseCategory`, `total` `Money`).

`apps/api/routes.go` maps the operation to `report:read` in `operationPermissions`.

## 4. Data spec

- **Schema:** no migration for F-10/01. F-10/02 adds an index migration only if the index check (§1.8) needs one.
- **Queries (sqlc, `internal/report/queries.sql`):**
  - `CountClaimsByStatus`.
  - `ApprovedTotalsByMonthCategory`, which groups on `to_char(decided_at AT TIME ZONE 'Asia/Bangkok', 'YYYY-MM')` and `line_items.category`.
  - `AverageDecisionHours`.
  - Each takes the actor's ID and a `finance` flag for the scope, and Bangkok-midnight bounds computed in Go (half-open, as F-09 §1.8). The sqlc config adds the new query file.
- **Seed:** `bun run seed` does not change. `seed:load` is §1.8.

## 5. Acceptance criteria

| ID | Requirement | Acceptance criterion |
| --- | --- | --- |
| F10-AC-01 | FR-14 | `GET /reports/summary` as a reviewer covers only claims assigned to them. As a Finance Officer it covers every non-draft claim. Drafts are never counted. |
| F10-AC-02 | FR-14 | `countsByStatus` has the six non-draft keys, with `0` for empty statuses, and counts current statuses without the date range. |
| F10-AC-03 | FR-14 | `approvedTotals` gives the exact line-item sums of `APPROVED` claims by Bangkok month of `decided_at` and category, in §1.4 order, and only for claims decided in the range. |
| F10-AC-04 | FR-14 | `averageDecisionHours` is the mean of `decided_at − first_submitted_at` for `APPROVED` and `REJECTED` claims decided in the range, rounded to one decimal place, or `null` when there are none. |
| F10-AC-05 | FR-14 | `from` and `to` are optional, inclusive Bangkok dates on `decided_at`, with the §1.6 month-edge boundaries. An invalid date or `from` > `to` → 400. |
| F10-AC-06 | FR-14, NFR-02 | A requester → 403; no session → 401. The route inventory covers the operation. |
| F10-AC-07 | FR-14 | `internal/report` does not import `internal/claim`; the lint rule fails the build if it does. |
| F10-AC-08 | FR-14, NFR-08 | The Summary page shows the §2.1 tiles, the month × category table with `฿` amounts and `Oct 2026` months, and the average time. The date range is in the URL. The "Summary" header link shows only with `report:read`. |
| F10-AC-09 | FR-14, NFR-08 | With no decisions in the range, the page shows "No decisions in this date range." and "–" for the average. A requester who opens `/reports` sees the 403 page. |
| F10-AC-10 | NFR-07 | `seed:load` adds at least 200 claims spread over statuses, categories, and months, each with a valid history. A second run adds nothing. |
| F10-AC-11 | NFR-07 | The list, filter, and summary queries were checked with `EXPLAIN ANALYZE` on the load data, and the plans are recorded in the F-10/02 PR. |
| F10-AC-12 | NFR-07 | The timed run on the deployed instance with the load data has an overall p95 under 1 s. The result is saved with the tag and commit, with p95 per request type. |

## 6. Software Test Specification

| Test | AC | Level | Precondition / input | Action | Expected result |
| --- | --- | --- | --- | --- | --- |
| F10-TEST-01 | F10-AC-01 | Integration (module) | Fixed dataset: claims assigned to `rev.malee` and to `rev.somchai` in each status, one of `rev.malee`'s own claims, drafts | `Summary` as `rev.malee`; as `fin.wanida` | Only `rev.malee`'s assigned claims; every non-draft claim; no draft in either |
| F10-TEST-02 | F10-AC-02 | Integration (module) | Fixed dataset with no `RETURNED` claim in scope | `Summary` with a range that excludes every decision | Six keys, `RETURNED` `0`, counts equal the full scope |
| F10-TEST-03 | F10-AC-03 | Integration (module) | Fixed dataset: `APPROVED` claims in October and November, one with `TRAVEL` and `MEALS` lines, one `REJECTED` claim | `Summary` with no range; with October only | Exact sums per month × category in §1.4 order, the mixed claim in two entries, no `REJECTED` amounts; only the October entries |
| F10-TEST-04 | F10-AC-04 | Integration (module) | Decided claims with known durations of 2 h, 3 h, and 25 h 6 min; an empty range | `Summary` | `averageDecisionHours` 10.0 (30.1 h / 3 = 10.03 → 10.0); `null` for the empty range |
| F10-TEST-05 | F10-AC-05 | Integration (module) | Claims decided at `2026-10-31T23:59:59+07:00`, `2026-11-01T00:00:00+07:00`, `2026-10-31T17:30:00Z` | `Summary` with October; with only `from=2026-11-01` | First claim only; second and third, with the third in month `2026-11` |
| F10-TEST-06 | F10-AC-05, F10-AC-06 | Integration | Seed data | `GET /reports/summary` as `rev.malee` and `fin.wanida`; as `req.anan`; with no session; with `from=2026-11-10&to=2026-11-01`; with `from=Oct` | 200 with the §4.4 shape ×2; 403; 401; 400 ×2 with the F-09 messages |
| F10-TEST-07 | F10-AC-07 | Unit | `apps/api` | `bun run lint` with a test-only import of `internal/claim` in `internal/report` (run once by hand, result recorded in the PR) | `depguard` error; without the import, lint passes |
| F10-TEST-08 | F10-AC-08 | Component | MSW summary with two months and three categories; `/auth/me` with and without `report:read` | Render `/reports?from=2026-10-01&to=2026-11-30`; change **From**; render the header | Tiles and counts; table cells `฿18,250.00`, "–", month totals, `Oct 2026`; URL updates; "Summary" link only with the permission |
| F10-TEST-09 | F10-AC-09 | Component | MSW summary with `[]` and `null`; MSW 403 | Render `/reports` | "No decisions in this date range." and "–"; the 403 page |
| F10-TEST-10 | F10-AC-10 | Integration | Database with the base seed | Run `seed-load` twice; replay each load claim's history through the lifecycle table | ≥ 200 load claims after the first run and the same counts after the second; every non-draft status, every category, and five decision months present; every history is a valid path; line items pass the BR rules |
| F10-TEST-11 | F10-AC-11 | Manual | Load data on a local stack | `EXPLAIN ANALYZE` of each list, filter, and summary query | Plans recorded in the F-10/02 PR with the commit ID |
| F10-TEST-12 | F10-AC-12 | Manual | Deployed release with the base seed and the load data | `bun run perf -- --base-url <deployed URL> --rounds 20` | Warm-up round not counted; overall p95 < 1 s; output saved as `docs/perf/nfr-07-<tag>.md` with the tag and commit |
| F10-TEST-13 | F10-AC-08 | E2E | Seeded local stack with the load data | `rev.malee` opens Summary, picks October 2026 | The tiles, the table, and the average show for her assigned claims |

## 7. Feature Definition of Done

The Sprint 3 Product DoD applies. For F-10 the evidence is:

- [ ] All automated F10-TEST rows run: unit, integration, and component tests pass in CI on the merge commit; F10-TEST-13 passes on a local seeded stack with the tested commit recorded.
- [ ] TDD evidence for the scope (F10-TEST-01) and the month edges (F10-TEST-05): red and green commit IDs in the PR.
- [ ] `openapi.yaml` has `getDecisionSummary` and its schemas; both apps regenerate with no diff in CI. F01-TEST-12 is green with the new operation.
- [ ] F10-TEST-11 and F10-TEST-12 are recorded with the commit ID; F10-TEST-12 with the release tag on the deployed instance.
- [ ] `docs/traceability.md` lists the actual test names and results for the F-10 rows and the NFR-07 row.

## 8. Revision history

| Version | Date | Change | Author |
| --- | --- | --- | --- |
| 0.1 | 2026-11-25 | Initial contract, from the F-10 feature spec and tickets #49 and #50 | beambeambeam |
| 0.2 | 2026-11-25 | Review on #111: at least 60 load claims to approve; the timed run stops when too few are left and starts with an untimed warm-up round | beambeambeam |
