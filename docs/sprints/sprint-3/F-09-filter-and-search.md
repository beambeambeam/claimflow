# F-09 Filter and Search – Engineering Contract

- **Feature:** F-09 Filter and search (`docs/features.md`)
- **Sprint:** 3 · **Issue:** #48 (F-09/01)
- **Requirements:** FR-13, NFR-02, NFR-07, NFR-08
- **Responsibilities verified here:** SR-01 (new scope; the list and the policy agree on visibility), SR-05 (filter indexes)
- **Depends on:** F-03 (list and pagination), F-04 (review scope), F-08 (Finance visibility)
- **Design baseline:** `docs/system-sds.md` v0.18 §4.3 `GET /claims` filters; §5.2 visibility; §5.3 `scope=all`; §7.2 shared UI behaviour
- **Version:** 0.1 · **Last updated:** 2026-11-25

---

## 1. Feature SDS

### 1.1 Behavior

- Every signed-in user opens **All Claims**. It lists every claim they can see (SDS §5.2), newest first:
  - A requester sees their own claims, drafts included.
  - A reviewer sees their own claims and the claims assigned to them that are not `DRAFT`.
  - A Finance Officer sees every claim that is not `DRAFT`.
- The list can be narrowed by status, category, a date range, and a title search. All filters are optional and combine with AND.
- Filters never widen what a user can see. The list query and the claim policy use the same visibility rule, and a test checks that they agree for every seed user.
- Filters live in the URL, so a filtered view can be bookmarked and shared with someone who can see the same claims.

The F-09 ticket says "a reviewer sees only assigned claims". The SDS (§5.2, §5.3) and this contract include the reviewer's own claims too, because `scope=all` is the whole visible set.

### 1.2 Workflow

```mermaid
sequenceDiagram
    participant U as User
    participant W as Web (All Claims)
    participant H as Handler
    participant C as claim.Service.List
    participant D as PostgreSQL
    U->>W: choose status, category, dates; type a title
    W->>W: update URL search params (search debounced 300 ms)
    W->>H: GET /claims?scope=all&status=&category=&from=&to=&q=&page=
    H->>H: session; contract validation; filter rules (400)
    H->>C: List(actor, Query{ScopeAll, Filters, Page, PageSize})
    C->>D: SELECT … WHERE visible(actor) AND filters ORDER BY created_at DESC, id DESC LIMIT/OFFSET
    C->>D: SELECT count(*) with the same WHERE
    H-->>W: 200 ClaimPage
```

### 1.3 Filters

| Parameter | Rule | Invalid value |
| --- | --- | --- |
| `status` | One `ClaimStatus` value, matched exactly. `DRAFT` is allowed and matches only the actor's own drafts. | Not a `ClaimStatus` value → 400 |
| `category` | One `ExpenseCategory` value. A claim matches if any of its line items has it. A claim appears once even if several line items match. | Not an `ExpenseCategory` value → 400 |
| `from`, `to` | Dates `YYYY-MM-DD`, inclusive, in Asia/Bangkok. They compare against the date of `first_submitted_at`, or of `created_at` for a claim that was never submitted. Either may be given alone. | Not a valid date → 400; `from` later than `to` → 400 |
| `q` | Trimmed. A case-insensitive substring of the title. `%`, `_`, and `\` in `q` are literal characters, not wildcards. Empty after trimming means no title filter. | Longer than 120 characters (the title maximum) → 400 |
| `page`, `pageSize` | As F-03: `page` ≥ 1, `pageSize` 1–100, default 20. | 400 |

Filters apply only to `scope=all` (SDS §5.3). A filter parameter with `scope=mine`, `review`, or `finance` → 400, so a client never gets an unfiltered list it thinks is filtered.

Every 400 uses the problem shape with one entry in `errors` per bad parameter (`field` is the parameter name). Messages:

| Field | Message |
| --- | --- |
| `status` | "Unknown status." |
| `category` | "Unknown category." |
| `from`, `to` | "Use a date in the form YYYY-MM-DD." |
| `to` (from > to) | "The end date must be on or after the start date." |
| `q` | "Search text must be at most 120 characters." |
| any filter with another scope | "Filters are only available for scope=all." |

### 1.4 Visibility in SQL

The `all` query has one `WHERE` clause for visibility, written once in `queries.sql`:

`requester_id = actor OR (assigned_reviewer_id = actor AND status <> 'DRAFT') OR (actor_role = 'finance' AND status <> 'DRAFT')`

It is the SQL form of `isVisible` in `policy.go`. F09-TEST-08 checks that both give the same set for every seed user, so a change to one without the other fails a test.

### 1.5 Ordering

Newest first by `created_at`, then `id` (the `mine` order from F-03).

### 1.6 Failure and boundary cases

| Case | Result |
| --- | --- |
| `req.anan`, `scope=all`, no filters | Her claims, drafts included, newest first |
| `rev.malee`, `scope=all` | Her own claims and the non-draft claims assigned to her |
| `fin.wanida`, `scope=all&status=DRAFT` | 200, empty list |
| `category=MEALS` on a claim with a `MEALS` and a `TRAVEL` line | The claim appears once |
| `from=2026-11-05&to=2026-11-05`, claim first submitted at `2026-11-05T23:59:59+07:00` / `2026-11-06T00:00:00+07:00` | Included / excluded |
| Claim first submitted at `2026-11-04T17:30:00Z` (`2026-11-05` 00:30 in Bangkok), `from=2026-11-05` | Included |
| A returned and resubmitted claim | Filtered on `first_submitted_at`, not the latest `submitted_at` |
| A draft created on 5 Nov, `from=2026-11-05&to=2026-11-05` | Included (the owner only) |
| `q=TAXI` against the title "Taxi to client site" | Included |
| `q=100%` | Matches the literal text "100%" |
| `status=PAID`, `category=FOOD`, `from=05/11/2026`, `from=2026-11-10&to=2026-11-01`, 121-character `q` | 400 with the §1.3 messages |
| `scope=mine&status=APPROVED` | 400 |
| No session | 401 |

### 1.7 Permissions and policy

`scope=all` needs a session and no route permission (SDS §5.3). Finance has no `claim:list`, so `all` is the only list where a Finance Officer can browse the whole department. The claim policy does not change.

### 1.8 Code

- `Query` gains `Filters{Status, Category, From, To, Q}` with pointer or zero-value "not set" fields, and `ScopeAll`.
- `Actor.Role` (added by F-08) feeds the visibility clause.
- The filters are SQL parameters with `sqlc.narg`, so one query serves every combination. No SQL is built from strings.
- The date range is converted to Bangkok-midnight timestamps in Go before the query (`from` 00:00 to `to + 1 day` 00:00, half-open).
- The handler checks the §1.3 rules and returns all bad parameters in one 400.

## 2. UI spec

### 2.1 All Claims (`/all-claims`, `_auth/all-claims.tsx`)

The page lives in `features/claims`.

| Element | Behavior |
| --- | --- |
| Header link | "All Claims" for every signed-in user. |
| Filter bar | **Status** (All + the seven statuses with their badge labels), **Category** (All + the six categories), **From** and **To** date inputs, **Search title** text input, **Clear filters**. |
| URL | Typed router search params `status`, `category`, `from`, `to`, `q`, `page`. A filter change replaces the param and resets `page` to 1. Opening a URL with params fills the controls. Unknown values in the URL are dropped by the search-param schema. |
| Search | Debounced 300 ms after the last key press. |
| Table | Columns: Claim no., Requester, Title, Status badge, Total, Submitted (`5 Nov 2026`; "–" for a draft). Newest first. A row opens Claim Detail. |
| Pagination | As My Claims (F-03 §2.1), with the filters kept in the URL. |
| Clear filters | Removes every filter param and shows page 1. Hidden when no filter is set. |

### 2.2 States

| State | Behavior |
| --- | --- |
| Loading | 5 skeleton rows; the filter bar stays usable. |
| Empty, no filters | "No claims yet." |
| Empty, with filters | "No claims match these filters." and a **Clear filters** button. |
| 400 | Not expected from the UI: the controls only send valid values, and the web checks `from` ≤ `to` before it sends. If the API still returns 400, the field message shows under the control. |
| Error | As F-03 (inline error with **Try again**). |

## 3. API spec

| Method | Path | Request | Success | Errors | Auth |
| --- | --- | --- | --- | --- | --- |
| GET | `/claims?scope=all&status=&category=&from=&to=&q=&page=&pageSize=` | Query parameters as §1.3 | 200 `ClaimPage` | 400, 401 | session |

The `scope` enum gains `all` (`ListScopeAll`). The new query parameters are `status` (`ClaimStatus`), `category` (`ExpenseCategory`), `from` and `to` (`format: date`), and `q` (`maxLength: 120`). The contract validator handles enum, format, and length. The handler handles `from` > `to` and filters with another scope. The `listClaims` description lists the §1.3 rules.

## 4. Data spec

- **Schema:** migration `00003_claim_filters.sql` adds:
  - `line_items (category, claim_id)` for the category match.
  - `claims (status, created_at DESC)` for the status filter with the newest-first order.
  - No index for `q`. A substring match on titles of a few hundred claims is a scan of a small table. F-10/02 checks every list and filter query with `EXPLAIN` on the `seed:load` data and adds an index there if a query misses NFR-07.
- **Queries (sqlc):** `ListAllClaims` and `CountAllClaims`, with the §1.4 visibility clause and `sqlc.narg` filters. Category uses `EXISTS (SELECT 1 FROM line_items …)`, so a claim is never duplicated.
- **Seed:** no new rows. Tests build their own claims with fixed `created_at` and `first_submitted_at`.

## 5. Acceptance criteria

| ID | Requirement | Acceptance criterion |
| --- | --- | --- |
| F09-AC-01 | FR-13, NFR-02 | `GET /claims?scope=all` with a session returns every claim the actor can see (§1.1), newest first, paged as F-03. Without a session → 401. |
| F09-AC-02 | FR-13 | `status` returns only claims in that status. `category` returns claims with at least one line item in that category, each claim once. |
| F09-AC-03 | FR-13 | `from` and `to` are inclusive Asia/Bangkok dates on `first_submitted_at`, or `created_at` for a never-submitted claim. The Bangkok-midnight boundaries in §1.6 hold. |
| F09-AC-04 | FR-13 | `q` is a trimmed, case-insensitive substring match on the title, with `%`, `_`, and `\` matched literally. |
| F09-AC-05 | FR-13 | Filters combine with AND, and `total` counts the filtered set. |
| F09-AC-06 | FR-13, NFR-08 | Invalid filter values, `from` later than `to`, a `q` over 120 characters, or a filter with another scope → 400 with one `errors` entry per bad parameter and the §1.3 message. |
| F09-AC-07 | FR-13, NFR-02 | For each seed user, `scope=all` with no filters returns exactly the claims that `Allowed` marks visible. No filter makes a claim appear that is not visible. |
| F09-AC-08 | NFR-07 | The migration adds the §4 indexes. The filter queries use SQL parameters, not string-built SQL. |
| F09-AC-09 | FR-13, NFR-08 | The All Claims page has the §2.1 filter bar. Filters are in the URL; changing one resets the page to 1; opening a URL with filters fills the controls; search is debounced. |
| F09-AC-10 | FR-13, NFR-08 | With filters and no results, the page shows "No claims match these filters." and **Clear filters**, which removes every filter. Without filters and no claims, it shows "No claims yet." |
| F09-AC-11 | FR-13 | End to end: `rev.malee` opens All Claims, filters by status `APPROVED`, then searches part of a title, and sees only matching claims she can see; Clear filters brings the full list back. |

## 6. Software Test Specification

| Test | AC | Level | Precondition / input | Action | Expected result |
| --- | --- | --- | --- | --- | --- |
| F09-TEST-01 | F09-AC-01 | Integration | Seed data | `GET /claims?scope=all` as `req.anan`, `rev.malee`, `fin.wanida`; with no session | The §1.1 set for each, newest first, `total` matches; 401 |
| F09-TEST-02 | F09-AC-02 | Integration (module) | Claims in four statuses; a claim with `MEALS` and `TRAVEL` lines | `List` with `Status=APPROVED`; with `Category=MEALS`; with `Category=TRAVEL` | Only `APPROVED`; the mixed claim once; the mixed claim once |
| F09-TEST-03 | F09-AC-03 | Integration (module) | Claims first submitted at `2026-11-05T23:59:59+07:00`, `2026-11-06T00:00:00+07:00`, `2026-11-04T17:30:00Z`; a resubmitted claim with `first_submitted_at` 1 Nov and `submitted_at` 6 Nov; a draft created 5 Nov | `List` with `from=to=2026-11-05`; with only `from=2026-11-06`; with only `to=2026-11-05` | First, third, and draft; second only; first, third, resubmitted, and draft |
| F09-TEST-04 | F09-AC-04 | Integration (module) | Titles "Taxi to client site", "100% refund", "Hotel_A" | `List` with `q` = " TAXI ", "100%", "_", "%" | One match each for the first three; "%" matches only "100% refund" |
| F09-TEST-05 | F09-AC-05 | Integration (module) | Mixed claims | `List` with status, category, date range, and `q` together; with `PageSize=1` | Only claims that match all four; `total` is the filtered count, one row per page |
| F09-TEST-06 | F09-AC-06 | Integration | Session as `req.anan` | `scope=all` with `status=PAID`; `category=FOOD`; `from=05/11/2026`; `from=2026-11-10&to=2026-11-01`; a 121-character `q`; `status=PAID&category=FOOD`; `scope=mine&status=APPROVED` | 400 each with the §1.3 messages; two `errors` entries for the combined case |
| F09-TEST-07 | F09-AC-06 | Unit | Filter inputs | Handler filter check | Same results as F09-TEST-06 without HTTP; valid inputs give no errors |
| F09-TEST-08 | F09-AC-07 | Integration (module) | Seed data plus one claim of every status for each requester | For each seed user: `List(ScopeAll)` with no filters, and `Allowed` on every claim in the table | Same set of IDs; repeated with `Status` and `Category` filters, every listed claim is visible |
| F09-TEST-09 | F09-AC-08 | Integration | Migrated database | Read `pg_indexes` | Both §4 indexes exist |
| F09-TEST-10 | F09-AC-09 | Component | MSW list handler that records query params | Render `/all-claims?status=APPROVED&page=3`; change category; type "taxi" quickly | Controls filled from the URL; category change sets `page=1`; one request with `q=taxi` after 300 ms, not one per key |
| F09-TEST-11 | F09-AC-10 | Component | MSW: empty page with filters; empty page without filters | Render; click **Clear filters** | "No claims match these filters." and the button; URL has no filter params after the click; "No claims yet." with no button |
| F09-TEST-12 | F09-AC-09 | Component | `/auth/me` for each role | Render the header | "All Claims" link for every role |
| F09-TEST-13 | F09-AC-11 | E2E | Seeded database | §5 flow | Filtered rows match the status and title; the full list returns after Clear filters |

The route inventory test (F01-TEST-12) already covers `listClaims`.

## 7. Feature Definition of Done

The Sprint 3 Product DoD applies. For F-09 the evidence is:

- [ ] All F09-TEST rows run: unit, integration, and component tests pass in CI on the merge commit; F09-TEST-13 passes on a local seeded stack with the tested commit recorded.
- [ ] TDD evidence for the visibility agreement (F09-TEST-08) and the date boundaries (F09-TEST-03): red and green commit IDs in the PR.
- [ ] `openapi.yaml` has `scope=all` and the filter parameters; both apps regenerate with no diff in CI.
- [ ] Migration `00003_claim_filters.sql` runs up and down on a clean database.
- [ ] `docs/traceability.md` lists the actual test names and results for the F-09 rows.

## 8. Revision history

| Version | Date | Change | Author |
| --- | --- | --- | --- |
| 0.1 | 2026-11-25 | Initial contract, from the F-09 feature spec and ticket #48 | beambeambeam |
