# F-04 Review decision – Engineering Contract

- **Feature:** F-04 Review decision (`docs/features.md`)
- **Sprint:** 1 · **Issue:** #6
- **Requirements:** FR-05, FR-06, FR-07, BR-01, BR-02, BR-03, BR-04, BR-11, BR-13, NFR-02, NFR-06, NFR-08
- **Responsibilities verified here:** SR-03 (NFR-05, local restart), SR-04 (BR-03, BR-11, BR-13, NFR-06)
- **Depends on:** F-01, F-02, F-03 (claim policy and Claim Detail)
- **Design baseline:** `docs/system-sds.md` v0.8 §2.2 `claim_history`; §3.2 `APPROVE`, `REJECT`; §3.3; §4.1 versions; §4.3 `approve`, `reject`; §5.2; §5.3 `scope=review` and access tests; §7.2 decisions; ADR-0001
- **Version:** 0.2 (draft) · **Last updated:** 2026-11-06

---

## 1. Feature SDS

### 1.1 Behavior

- A Reviewer or Finance Officer opens the Review Queue and sees the `Submitted` claims assigned to them, oldest first.
- On Claim Detail, the assigned reviewer approves (optional comment) or rejects (required reason).
- The status change and its history entry are written in one transaction.
- Self-approval, decisions by unrelated reviewers, decisions on claims that are not `Submitted`, and decisions made from an out-of-date screen are refused.
- A Finance Officer reviews the Department Head's claims in the same way. In Sprint 1, `approve` always leads to `Approved`; the two-stage rule comes in F-08 (Sprint 3).

### 1.2 Workflow

```mermaid
sequenceDiagram
    participant V as Assigned reviewer
    participant W as Web (Claim Detail)
    participant H as Handler
    participant C as claim.Service.Act
    participant D as PostgreSQL
    V->>W: Approve / Reject (+ comment)
    W->>H: POST /claims/{id}/approve, If-Match: "3"
    H->>H: If-Match present? (else 428)
    H->>C: Act(actor, id, version 3, APPROVE, comment)
    C->>D: BEGIN; load claim
    C->>C: policy: visible? (403) allowed action? (403 / 409 BR-03)
    C->>C: BR-04 comment rule (422)
    C->>D: UPDATE claims SET status, version+1, decided_at WHERE id AND version = 3
    alt 0 rows
        C->>D: ROLLBACK → 409 CONFLICT
    else 1 row
        C->>D: INSERT claim_history; COMMIT
        C-->>H: Claim (version 4)
        H-->>W: 200 + ETag "4"
    end
```

### 1.3 Lifecycle (Sprint 1)

| From | Action | To | Who | History |
| --- | --- | --- | --- | --- |
| `SUBMITTED` | `APPROVE` | `APPROVED`, `decided_at = now` | Assigned reviewer, not the owner | `APPROVE`, comment optional |
| `SUBMITTED` | `REJECT` | `REJECTED`, `decided_at = now` | Assigned reviewer, not the owner | `REJECT`, reason required |

`APPROVED` and `REJECTED` are terminal. Any other transition → `ErrInvalidTransition{BR-03}`.

### 1.4 Order of checks (SDS §3.3)

| Step | Failure | Response |
| --- | --- | --- |
| 1. `If-Match` header | missing / malformed | 428 `PRECONDITION_REQUIRED` / 400 |
| 2. Claim exists | no | 404 `NOT_FOUND` |
| 3. Claim visible to actor | no | 403 `FORBIDDEN` (no claim data) |
| 4. Actor relation allows the action | own (BR-01), not assigned (BR-02) | 403 `FORBIDDEN` |
| 5. Status allows the action | not `SUBMITTED` (BR-03) | 409 `INVALID_TRANSITION`, `errors[0].rule = "BR-03"` |
| 6. Comment rule | BR-04 | 422 `VALIDATION_FAILED`, `field = "comment"` |
| 7. Conditional update | version changed (BR-13) | 409 `CONFLICT` |
| 8. History insert, commit | database error (NFR-06) | 500; whole transaction rolled back |

Steps 4 and 5 both come from the policy: when the actor's relation is wrong, the response is 403 whatever the status.

### 1.5 Comment rule (BR-04)

- The comment is trimmed. Length is counted in Unicode characters (Thai text counts per character, not per byte).
- `reject`: required, 5–500 characters.
- `approve`: optional; empty or blank is stored as NULL; if present, at most 500 characters.

| Input | Message |
| --- | --- |
| Missing or blank (reject) | "Enter a reason (5 to 500 characters)." |
| 1–4 characters (reject) | "Reason must be at least 5 characters." |
| More than 500 (reject or approve) | "Comment cannot be more than 500 characters." |

### 1.6 Failure and boundary cases

| Case | Result |
| --- | --- |
| Reject reason of 4 / 5 / 500 / 501 characters | 422 / 200 / 200 / 422 |
| `rev.malee` approves her own claim (assigned to `rev.prasert`) | 403 (BR-01) |
| Policy called with a claim whose `assigned_reviewer_id = requester_id` | No `approve` or `reject` in actions (BR-01). The database CHECK also refuses to store such a row. |
| `rev.somchai` approves `req.anan`'s claim (assigned to `rev.malee`) | 403 (BR-02) |
| `req.anan` approves his own claim | 403 |
| Approve an `APPROVED` or `REJECTED` claim | 409 `INVALID_TRANSITION` BR-03; history unchanged |
| Two decisions at once with the same version | One 200, one 409 `CONFLICT`; one history row |
| `If-Match` with a well-formed version that is not the current one | 409 `CONFLICT` |
| No `If-Match` | 428 |
| History insert fails | 500; status, version, `decided_at` unchanged |
| `UPDATE` or `DELETE` on `claim_history` | Database error (trigger) |
| `fin.wanida` approves `rev.prasert`'s 40,000.00 claim | 200, `APPROVED` (ADR-0001) |

### 1.7 Permissions

| Operation | Anonymous | `requester` | `reviewer` | `finance` |
| --- | --- | --- | --- | --- |
| `GET /claims?scope=review` | 401 | 403 | claims assigned to them | claims assigned to them |
| `POST /claims/{id}/approve`, `/reject` | 401 | 403 | assigned and not own | assigned and not own |

## 2. UI spec

### 2.1 Review Queue (`/reviews`, `_auth/reviews.tsx`)

| Element | Behavior |
| --- | --- |
| Table | Columns: Claim no., Requester, Title, Total, Submitted. Oldest first. A row opens Claim Detail. |
| Header link | "Review Queue" when `review:list` is in `permissions` (F-01). |
| Pagination | As My Claims (F-03 §2.1). |

### 2.2 Decision panel on Claim Detail

The route file `_auth/claims/$claimId.tsx` renders `DecisionPanel` from `features/reviews` when `allowedActions` contains `approve` or `reject`. `features/claims` and `features/reviews` do not import each other.

| Control | Behavior |
| --- | --- |
| **Approve** | Opens a confirmation dialog: "Approve claim CLM-…?", optional comment, **Approve** and **Cancel**. |
| **Reject** | Opens a dialog with a required reason field, a counter (`0 / 500`), the field messages from §1.5, **Reject** and **Cancel**. |
| Sending | Dialog buttons disabled with a spinner. |
| Success | Toast "Claim approved." or "Claim rejected."; the claim detail and the queue are refetched; the panel disappears because `allowedActions` is empty. |
| 409 | Toast "This claim was changed by someone else – reloaded", then refetch. |
| 422 | Message under the reason field; dialog stays open with the text kept. |
| 403 | "You don't have access to this claim." page. |

### 2.3 States

| State | Review Queue |
| --- | --- |
| Loading | 5 skeleton rows |
| Empty | "No claims waiting for your review." |
| 403 | "You don't have access to this page." (requester opening `/reviews` directly) |
| 5xx / network | Inline error with **Try again** and `requestId` |

The requester sees a reject reason in the history timeline on Claim Detail (F-03 §2.2).

## 3. API spec

| Method | Path | Request | Success | Errors | Auth |
| --- | --- | --- | --- | --- | --- |
| GET | `/claims` | `scope=review`, `page`, `pageSize` | 200 `ClaimPage`; status `SUBMITTED`, `assigned_reviewer_id = actor`, `submittedAt` ascending | 400, 401, 403 | `review:list` |
| POST | `/claims/{id}/approve` | Header `If-Match: "<version>"`; body `{ "comment"?: string }` | 200 `{ "data": Claim }`, new `ETag` | 400, 401, 403, 404, 409 `INVALID_TRANSITION` / `CONFLICT`, 422, 428 | session; policy |
| POST | `/claims/{id}/reject` | Header `If-Match: "<version>"`; body `{ "comment"?: string }` | 200 `{ "data": Claim }`, new `ETag` | same | session; policy |

`If-Match` is an optional header in `openapi.yaml` (`required: false`), so the validator never rejects a request without it. The handler returns 428 `PRECONDITION_REQUIRED` when it is missing and 400 when it is not a quoted version number (§1.4 step 1). This applies to `approve` and `reject`, and to every later operation that changes a claim.

`comment` is optional on both operations and has no `minLength` or `maxLength` in `openapi.yaml`. The BR-04 check (§1.5) decides the response, so a missing, blank, too short, or too long reason gets 422 with `field = "comment"`, never a 400 from the validator.

`Claim` and `ClaimPage` are defined in F-02 §3.1 and F-03 §3.1. After a decision, `allowedActions` for the reviewer is `[]`.

Example 409 for an invalid transition:

```json
{
  "type": "about:blank",
  "title": "This action is not allowed in the claim's current status.",
  "status": 409,
  "code": "INVALID_TRANSITION",
  "instance": "/api/v1/claims/0192f1c4-.../approve",
  "requestId": "019a5e2c-7b3d-7c41-9f2e-4b8d1a6c3e70",
  "errors": [
    {
      "field": "status",
      "rule": "BR-03",
      "message": "Only a submitted claim can be approved or rejected."
    }
  ]
}
```

## 4. Data spec

- **Schema:** no change. F-04 uses `claims.status`, `version`, `decided_at`, and `claim_history` from F-02 `00002_claims.sql`, including the append-only trigger (BR-11).
- **Seed:** `bun run seed` adds sample claims, idempotent by `claim_no`, so every Sprint 1 screen has data:

| Requester | Assigned reviewer | Status | History |
| --- | --- | --- | --- |
| `req.anan` | `rev.malee` | `SUBMITTED` | `SUBMIT` |
| `req.anan` | `rev.malee` | `APPROVED` | `SUBMIT`, `APPROVE` |
| `req.ploy` | `rev.malee` | `REJECTED` | `SUBMIT`, `REJECT` with reason |
| `req.niran` | `rev.somchai` | `SUBMITTED` | `SUBMIT` |
| `rev.malee` | `rev.prasert` | `SUBMITTED` | `SUBMIT` |
| `rev.prasert` | `fin.wanida` | `SUBMITTED` (40,000.00) | `SUBMIT` |

## 5. Acceptance criteria

| ID | Requirement | Acceptance criterion |
| --- | --- | --- |
| F04-AC-01 | FR-05 | `rev.malee` lists `scope=review` → only `SUBMITTED` claims assigned to her, oldest `submittedAt` first. Decided claims, her own claims, and claims assigned to other reviewers are not in the list. |
| F04-AC-02 | FR-05, NFR-02 | `req.anan` lists `scope=review` → 403. `fin.wanida` lists `scope=review` → only claims assigned to her (`rev.prasert`'s). |
| F04-AC-03 | FR-05 | The assigned reviewer opens a `SUBMITTED` claim → 200 with full history and `allowedActions = ["approve", "reject"]`. |
| F04-AC-04 | FR-06, FR-07, NFR-06 | `rev.malee` approves `req.anan`'s `SUBMITTED` claim with the current `If-Match` → 200: status `APPROVED`, `decidedAt` = now, `version` + 1, new `ETag`, and one new history entry (`APPROVE`, `SUBMITTED` → `APPROVED`, actor `rev.malee`, the comment or `null`). The claim leaves her queue. |
| F04-AC-05 | FR-06, FR-07, BR-04 | `rev.malee` rejects with a valid reason → 200: status `REJECTED`, `decidedAt` set, one `REJECT` history entry with the reason. `req.anan` sees the reason in the claim's history. |
| F04-AC-06 | BR-04, NFR-08 | Reject with a missing or blank reason, or 4 or 501 characters → 422 with `field = "comment"`, `rule = "BR-04"`, and the message from §1.5. 5 and 500 characters → 200. Approve with a 501-character comment → 422. |
| F04-AC-07 | BR-01 | `rev.malee` approves or rejects her own claim → 403, and status and history are unchanged. The policy gives no `approve` or `reject` when `assigned_reviewer_id = requester_id`, and the database refuses to store such a claim. |
| F04-AC-08 | BR-02, NFR-02 | `rev.somchai` approves or rejects `req.anan`'s claim → 403 with no claim data; status and history are unchanged. |
| F04-AC-09 | BR-02, NFR-02 | `req.anan` calls approve on his own claim → 403. `req.ploy` calls approve on `req.anan`'s claim → 403. |
| F04-AC-10 | FR-06, BR-02 | `fin.wanida` approves `rev.prasert`'s 40,000.00 claim → 200 with status `APPROVED` (ADR-0001; single stage in Sprint 1). |
| F04-AC-11 | BR-03 | The assigned reviewer approves or rejects an `APPROVED` or `REJECTED` claim → 409 `INVALID_TRANSITION` with `errors[0].rule = "BR-03"`. Status, `version`, and history are unchanged. |
| F04-AC-12 | BR-13 | Two decisions on the same claim at the same time with the same version → exactly one 200 and one 409 `CONFLICT`, and exactly one decision history row. |
| F04-AC-13 | BR-13 | A decision with a well-formed `If-Match` that is not the current version → 409 `CONFLICT`. Without `If-Match` → 428. With a malformed `If-Match` → 400. |
| F04-AC-14 | NFR-06 | If the history insert fails, the decision returns 500 and the claim's status, `version`, and `decidedAt` are unchanged. |
| F04-AC-15 | BR-11 | `UPDATE` and `DELETE` on any `claim_history` row fail with a database error. Deleting a claim that has history fails. |
| F04-AC-16 | NFR-05 | After the API process and the PostgreSQL container restart (volume kept), every claim, line item, and history entry created before the restart is returned unchanged. |
| F04-AC-17 | FR-05 | Review Queue shows the columns in §2.1, skeleton rows while loading, "No claims waiting for your review." when empty, and an inline error with **Try again**. A requester who opens `/reviews` sees the 403 page. |
| F04-AC-18 | FR-06, NFR-08 | The decision panel appears only when `allowedActions` has a decision. Approve asks for confirmation; Reject needs a reason and shows the field message. Success shows a toast and refetches; 409 shows "This claim was changed by someone else – reloaded" and refetches. |
| F04-AC-19 | FR-05 | `bun run seed` creates the sample claims in §4 with their history. Running it twice gives the same rows. |
| F04-AC-20 | FR-06, FR-07 | End to end: `req.anan` submits a claim, `rev.malee` approves it from the Review Queue, and `req.anan` sees `Approved` with the `SUBMIT` and `APPROVE` entries in history. |

## 6. Software Test Specification

Levels as in F-01 §6. Module tests call `claim.Service.Act` against `pgtest` PostgreSQL with a fixed clock.

| Test | AC | Level | Precondition / input | Action | Expected result |
| --- | --- | --- | --- | --- | --- |
| F04-TEST-01 | F04-AC-01 | Integration | Seed claims plus a second `SUBMITTED` claim for `rev.malee` submitted later | `GET /claims?scope=review` as `rev.malee` | Only her assigned `SUBMITTED` claims, earliest `submittedAt` first; no `APPROVED`, `REJECTED`, own, or `rev.somchai` claims |
| F04-TEST-02 | F04-AC-02 | Integration | Seed data | `scope=review` as `req.anan`; as `fin.wanida` | 403; only `rev.prasert`'s claim |
| F04-TEST-03 | F04-AC-03 | Integration | `req.anan`'s `SUBMITTED` claim | `GET /claims/{id}` as `rev.malee` | 200; `allowedActions = ["approve", "reject"]`; history present |
| F04-TEST-04 | F04-AC-04 | Integration | Same claim, version `v` | `POST …/approve` with `If-Match: "v"` and comment "OK" | 200; status `APPROVED`; `decidedAt` = clock; `version = v + 1`; `ETag: "v+1"`; 2 history rows, last is `APPROVE` with comment "OK"; claim not in `scope=review` |
| F04-TEST-05 | F04-AC-05 | Integration | Same claim | `POST …/reject` with reason "Missing receipt" | 200; `REJECTED`; last history `REJECT` with the reason; `GET` as `req.anan` shows it |
| F04-TEST-06 | F04-AC-06 | Integration (module) | `SUBMITTED` claim | `Act(REJECT)` with no comment, `"   "`, 4 chars, 5 chars, 500 chars, 501 chars; `Act(APPROVE)` with 501 chars | `ValidationError{comment, BR-04}` ×3, ok, ok, BR-04 error; BR-04 error |
| F04-TEST-07 | F04-AC-06 | Integration | `SUBMITTED` claim | `POST …/reject` with no `comment`; with `"   "`; with a 4-character reason | 422 each; `errors[0]` = `comment`, BR-04, with "Enter a reason (5 to 500 characters).", "Enter a reason (5 to 500 characters).", and "Reason must be at least 5 characters." |
| F04-TEST-08 | F04-AC-07 | Integration | `rev.malee`'s own `SUBMITTED` claim (assigned to `rev.prasert`) | `POST …/approve` and `…/reject` as `rev.malee` | 403 for both; status `SUBMITTED`; history length unchanged |
| F04-TEST-09 | F04-AC-07 | Unit | In-memory claim with `requester_id = assigned_reviewer_id = rev.malee`, `SUBMITTED` | `Allowed(rev.malee, claim)` | `approve` and `reject` not in actions |
| F04-TEST-10 | F04-AC-07 | Integration | – | `INSERT` a claim with `assigned_reviewer_id = requester_id` | CHECK constraint error |
| F04-TEST-11 | F04-AC-08 | Integration | `req.anan`'s claim assigned to `rev.malee` | `POST …/approve` and `…/reject` as `rev.somchai` | 403; body has no claim data; status and history unchanged |
| F04-TEST-12 | F04-AC-09 | Integration | `req.anan`'s `SUBMITTED` claim | `POST …/approve` as `req.anan`; as `req.ploy` | 403; 403 |
| F04-TEST-13 | F04-AC-10 | Integration | `rev.prasert`'s 40,000.00 `SUBMITTED` claim | `POST …/approve` as `fin.wanida` | 200; status `APPROVED` |
| F04-TEST-14 | F04-AC-11 | Integration | One `APPROVED` and one `REJECTED` claim assigned to `rev.malee` | `POST …/approve` and `…/reject` on each with the current version | 409 `INVALID_TRANSITION`, `errors[0].rule = "BR-03"`; `version` and history count unchanged |
| F04-TEST-15 | F04-AC-12 | Integration (module) | `SUBMITTED` claim, version `v` | Two goroutines start together: `Act(APPROVE, v)` and `Act(REJECT, v, valid reason)` | Exactly one success and one `ErrConflict`; one decision history row; final status matches the winner |
| F04-TEST-16 | F04-AC-13 | Integration | `SUBMITTED` claim at version `v` | `approve` with `If-Match: "v+1"` (`"1"` for a new claim); with no header; with `If-Match: abc` | 409 `CONFLICT`; 428 `PRECONDITION_REQUIRED`; 400 |
| F04-TEST-17 | F04-AC-14 | Integration (module) | Test-only trigger that raises on `INSERT` into `claim_history` | `Act(APPROVE)` | Error; claim status `SUBMITTED`, `version` and `decided_at` unchanged |
| F04-TEST-18 | F04-AC-15 | Integration | Claim with 2 history rows | `UPDATE claim_history SET comment = 'x'`; `DELETE FROM claim_history`; `DELETE FROM claims WHERE id = …` | Each statement fails; rows unchanged |
| F04-TEST-19 | F04-AC-16 | Manual | `docker compose` database with a named volume; seed plus one claim created and approved through the UI | Stop the API and the database container; start both; open My Claims and the claim as `req.anan` | Same claims, items, totals, and history as before the restart; result recorded with the commit |
| F04-TEST-20 | F04-AC-17 | Component | MSW: queue with 2 claims; empty; 500; `/auth/me` as requester | Render Review Queue for each | Columns and order; empty message; error with **Try again**; requester sees the 403 page |
| F04-TEST-21 | F04-AC-18 | Component | MSW claim with `allowedActions` `[]` and `["approve", "reject"]`; approve 200; reject 422; approve 409 | Render; open each dialog and submit | No panel for `[]`; confirm dialog; reason message under the field with text kept; toast on success; 409 toast and refetch |
| F04-TEST-22 | F04-AC-19 | Integration | Empty database | Run seed twice | Claims in §4 with their history exist after the first run; row counts are the same after the second |
| F04-TEST-23 | F04-AC-20 | E2E | Seeded database | Log in as `req.anan` and submit; log in as `rev.malee`, open the claim from Review Queue, approve; log in as `req.anan`, open the claim | Status `Approved`; history shows `SUBMIT` then `APPROVE` with actor names |
| F04-TEST-24 | F04-AC-05, F04-AC-18 | E2E | Seeded `SUBMITTED` claim | `rev.malee` rejects with "Receipt missing"; `req.anan` opens the claim | Status `Rejected`; reason shown in history |

## 7. Feature Definition of Done

The Product DoD applies. For F-04 the evidence is:

- [ ] All F04-TEST rows run; automated ones pass in CI on the merge commit; F04-TEST-19 is recorded with the commit.
- [ ] TDD evidence for BR-01 or BR-13: commit IDs of the failing test and the passing implementation, for #4.
- [ ] The Playwright journey (F04-TEST-23) runs in CI or is recorded with the tested commit.
- [ ] `docs/traceability.md` lists the actual test names and results for F-04 rows.

## 8. Revision history

| Version | Date | Change | Author |
| --- | --- | --- | --- |
| 0.1 | 2026-11-05 | Initial contract | 1005, 1045 |
| 0.2 | 2026-11-06 | Review fix (PR #23): reject `comment` is optional in the contract so BR-04 returns 422; F04-TEST-07 sends missing and blank reasons; `If-Match` is optional in `openapi.yaml`; the handler returns 428 when it is missing and 400 when it is malformed; F04-TEST-16 uses `If-Match: "v+1"`, because `"v-1"` is `"-1"` for a new claim | 1005, 1045 |
| 0.3 | 2026-11-07 | Problem example uses `type: about:blank` and a UUIDv7 `requestId`, matching SDS §4.2 v0.10 | 1005, 1045 |
