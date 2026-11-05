# F-03: My claims

Status: ready-for-agent Sprint: 1 Requirements: FR-04 Verifies: SR-01 (new endpoints) Depends on: F-01, F-02 Design: SDS §4.3 `GET /claims?scope=mine`, `GET /claims/{id}`; §5.2 visibility and allowed actions; §5.3 list scopes; §7.1 My Claims, Claim Detail; §7.2

## Problem Statement

After a requester submits a claim, they cannot see what happened to it. With paper and email they ask around to find out if it was approved. They also need to be sure that nobody else can read their claims.

## Solution

A requester opens My Claims and sees their own claims with claim number, title, total, and status. Opening a claim shows its line items, total, status, and full history timeline. A member who is not allowed to see a claim gets an access message with no claim data. The claim detail also carries the server-computed allowed actions for the current member, so later features show buttons without re-deriving rules.

## User Stories

1. As a requester, I want to list my own claims, so that I can track them (FR-04).
2. As a requester, I want each row to show claim number, title, total, status, and submission date, so that I can find a claim at a glance.
3. As a requester, I want the newest claims first, so that recent work is on top.
4. As a requester, I want the list to page when I have many claims, so that it loads fast.
5. As a requester, I want a status badge with a clear colour, so that I can scan statuses quickly.
6. As a requester, I want money shown as `฿1,250.00` and dates as `31 Oct 2026` in Bangkok time, so that values read naturally (AS-06).
7. As a requester with no claims, I want a friendly empty state with a "Create claim" action, so that I know what to do next.
8. As a requester, I want skeleton rows while the list loads, so that the page does not jump.
9. As a requester, I want an inline error with Try again when the list fails to load, so that I can recover without reloading.
10. As a requester, I want to open a claim and see its line items, so that I can check what I submitted.
11. As a requester, I want to see the server total on the detail page, so that I know what I will be paid.
12. As a requester, I want to see the claim's history as a timeline with action, actor, time, and comment, so that I know who did what (FR-07).
13. As a requester, I want to see my assigned reviewer's name, so that I know who is deciding.
14. As a reviewer, I want to see my own claims in My Claims, so that I track them like any requester.
15. As a requester, I want my claims hidden from other requesters, so that my expenses stay private.
16. As a requester, I want a clear "You don't have access to this claim" page when I open a claim that is not mine, so that I am not confused.
17. As a member, I want a "Claim not found" page for an ID that does not exist, so that I know the link is wrong.
18. As a security reviewer, I want a 403 response to contain no claim data, so that access errors leak nothing.
19. As a security reviewer, I want reviewers in the reporting line who are not the assigned reviewer to be unable to see the claim, so that visibility follows assignment only (Visible Claim).
20. As a Finance Officer, I want to be blocked from My Claims, so that I am not shown a list I can never fill (BR-15).
21. As a developer, I want claim detail to include the allowed actions for the current member, so that the web shows only buttons the server allows (SD-16).
22. As a developer, I want visibility and allowed actions decided by one pure policy function, so that list, detail, and actions agree.
23. As a developer, I want claim responses to carry a version and an `ETag`, so that later changes can use `If-Match`.

## Implementation Decisions

- **Claim policy.** A pure function `Allowed(actor, claim) → (visible, actions)`. In Sprint 1: visible if own, or assigned and not `DRAFT`. Actions in Sprint 1: `approve` and `reject` when assigned, not own, and `SUBMITTED` (used by F-04). Later features add rows to the same function. The "not own" check uses `requester_id` directly.
- **Claim module.** `Get(actor, id)` loads the claim with line items and history; `ErrNotFound` if missing, `ErrForbidden` if not visible. `List(actor, query)` with scope `mine`: own claims, all statuses, newest first, paginated.
- **API.** `GET /claims?scope=mine&page=&pageSize=` (route permission `claim:list`; pageSize max 100) returns `{ data, page, pageSize, total }`. `GET /claims/{id}` (session; policy) returns `{ data: Claim }` with line items, history, reviewer summary, `version`, and `allowedActions`, plus an `ETag` header. 403 and 404 use the problem format with no claim data.
- **Web.** My Claims route under `_auth` with a loader that ensures the list query; Claim Detail route that ensures the detail query. Shared presentational components for status badge, money, and dates (Asia/Bangkok). Inline 403 and 404 pages. Empty, loading, and error states follow SDS §7.2.

## Testing Decisions

- Test through public interfaces only.
- **Policy matrix (table-driven, in-process):** role × own/assigned/unrelated × status → visible and actions. Includes the self-approval case where assignment points at the requester.
- **Claim module against real PostgreSQL:** `List` scope `mine` returns only own claims; `Get` for another requester's claim → `ErrForbidden`; unrelated reviewer → `ErrForbidden`; missing ID → `ErrNotFound`.
- **HTTP:** `req.ploy` reads `req.anan`'s claim → 403 with no claim fields in the body; Finance `scope=mine` → 403; pagination shape; `ETag` matches `version`.
- **Web:** list rows, empty state, error with retry, detail with history, 403/404 pages.
- Prior art: F-01 HTTP tests and F-02 claim module tests.

## Out of Scope

- Review Queue (`scope=review`) and decision buttons (F-04).
- Filters and search (F-09).
- Finance department-wide visibility (F-08 and F-09, Sprint 3).

## Further Notes

- The policy function becomes the single place every later feature extends. Keep its tests table-driven so adding a row is cheap.
