# F-02 Submit claim – Engineering Contract

- **Feature:** F-02 Submit claim (`docs/features.md`)
- **Sprint:** 1 · **Issue:** #6
- **Requirements:** FR-02, FR-03, FR-07, BR-05, BR-06, BR-07, BR-08, BR-12, BR-15, BR-17, NFR-06, NFR-08
- **Depends on:** F-01
- **Design baseline:** `docs/system-sds.md` v0.8 §2.2 `claims`, `line_items`, `claim_history`; §2.3 money; §2.4 time; §3; §4.3 `POST /claims`; §5.2; §7.1 New Claim
- **Version:** 0.2 (draft) · **Last updated:** 2026-11-06

---

## 1. Feature SDS

### 1.1 Behavior

- A requester fills in a title, a purpose, and 1–20 line items, then submits the claim.
- The server checks every rule, calculates the total, records the requester's assigned reviewer, gives the claim a claim number, sets the status to `Submitted`, and writes the first history entry. All of this is one transaction.
- A member with no assigned reviewer cannot create a claim. In the seed data that is every Finance Officer.
- Saving as a draft is F-05 (Sprint 2). In Sprint 1, `POST /claims` always submits.

### 1.2 Workflow

```mermaid
sequenceDiagram
    participant R as Requester
    participant W as Web (New Claim)
    participant H as Handler
    participant C as claim.Service
    participant D as PostgreSQL
    R->>W: fill form, Submit
    W->>W: shape checks (generated Zod)
    W->>H: POST /claims
    H->>H: session, contract validation, route permission claim:create
    H->>C: Create(actor, input, submit = true)
    C->>C: BR-15, then BR-05, BR-06, BR-07, BR-12; total = Σ items
    C->>D: BEGIN; insert claim (SUBMITTED), line items, SUBMIT history; COMMIT
    C-->>H: Claim
    H-->>W: 201 Claim + ETag + Location
    W->>R: confirmation with claim number and status
```

### 1.3 Business rules

| Rule | Check | Where |
| --- | --- | --- |
| BR-05 | 1–20 line items. Each amount 0.01–20,000.00 THB. | `claim/rules.go` (422) |
| BR-05 | At most 2 decimal places. | Contract money pattern `^\d{1,6}\.\d{2}$` (400) |
| BR-06 | Total ≤ 50,000.00 THB. | `claim/rules.go` (422) |
| BR-07 | Each expense date ≤ today and ≥ (first submission date − 90 days), Asia/Bangkok dates. At first submission, the first submission date is today. | `claim/rules.go` with the injected clock (422) |
| BR-08 | Total = Σ line-item amounts, calculated on the server. Any `total` in the request is ignored. | `claim/service.go` |
| BR-12 | Category is one of `TRAVEL`, `ACCOMMODATION`, `MEALS`, `SUPPLIES`, `TRAINING`, `OTHER` (contract enum, 400). A `MEALS` item ≤ 1,500.00 THB (422). | Contract + `claim/rules.go` |
| BR-15 | The actor must have a `reviewer_id`. | `claim:create` route permission + `claim.Service` |
| BR-17 | `assigned_reviewer_id` is copied from the requester's `reviewer_id` at first submission and never changed afterwards. | `claim.Service` |
| FR-07 | One history row: action `SUBMIT`, from `NULL`, to `SUBMITTED`, actor = requester. | `claim.Service` |

Limits are named constants in `internal/claim/rules.go`. Business limits are not in the contract schema, so every limit error comes back as a 422 with its rule ID (NFR-08). The contract checks only shape: types, required fields, string lengths, the money format, the category enum, and the date format.

### 1.4 Validation messages (NFR-08)

| Field | Rule | Code path | Message |
| --- | --- | --- | --- |
| `items` | BR-05 count | 422 | "A claim needs 1 to 20 line items." |
| `items[i].amount` | BR-05 range | 422 | "Amount must be between 0.01 and 20,000.00 THB." |
| `items[i].amount` | BR-05 format | 400 / client | "Enter an amount with 2 decimals, e.g. 1250.00." |
| `total` | BR-06 | 422 | "Claim total cannot be more than 50,000.00 THB." |
| `items[i].amount` | BR-12 Meals | 422 | "A Meals item cannot be more than 1,500.00 THB." |
| `items[i].category` | BR-12 enum | 400 / client | "Choose a category." |
| `items[i].expenseDate` | BR-07 future | 422 | "Expense date cannot be in the future." |
| `items[i].expenseDate` | BR-07 window | 422 | "Expense date cannot be more than 90 days before the claim was first submitted." |
| `title` | shape | 400 / client | "Enter a title (up to 120 characters)." |
| `purpose` | shape | 400 / client | "Enter a purpose (up to 1,000 characters)." |
| `items[i].description` | shape | 400 / client | "Enter a description (up to 200 characters)." |
| `items[i].receiptRef` | shape | 400 / client | "Enter the receipt reference (up to 50 characters)." |

### 1.5 Failure and boundary cases

Fixed clock for the date cases: `2026-11-05T10:00:00+07:00`.

| Case | Input | Result |
| --- | --- | --- |
| Item count | 0 / 1 / 20 / 21 items | 422 / 201 / 201 / 422 (`items`, BR-05) |
| Amount | `"0.00"` / `"0.01"` / `"20000.00"` / `"20000.01"` | 422 / 201 / 201 / 422 (BR-05) |
| Decimals | `"1.005"` | 400 (`items[0].amount`) |
| Total | `20000.00 + 20000.00 + 10000.00` / `20000.00 + 20000.00 + 10000.01` | 201 / 422 (`total`, BR-06) |
| Meals | `MEALS "1500.00"` / `MEALS "1500.01"` / `TRAVEL "1500.01"` | 201 / 422 (BR-12) / 201 |
| Category | `"FOOD"` | 400 (`items[0].category`) |
| Expense date | `2026-11-06` / `2026-11-05` / `2026-08-07` (90 days) / `2026-08-06` (91 days) | 422 / 201 / 201 / 422 (BR-07) |
| Bangkok midnight | Clock `2026-11-05T17:30:00Z` (= 6 Nov 00:30 Bangkok); date `2026-11-06` | 201 |
| Client total | `"total": "1.00"` with items summing 1,250.00 | 201, `total = "1250.00"` |
| No assigned reviewer | `fin.wanida` | 403, no claim stored |
| History insert fails | Forced failure | 500; no claim, line item, or history row stored |

### 1.6 Permissions

| Actor | `POST /claims` |
| --- | --- |
| Anonymous | 401 |
| `requester`, `reviewer` with an assigned reviewer | 201 |
| `finance` (no `claim:create`) | 403 |
| Any actor with `reviewer_id` NULL | 403 (`ErrForbidden`, BR-15) |

## 2. UI spec

### 2.1 New Claim (`/claims/new`, `_auth/claims/new.tsx`)

| Element | Behavior |
| --- | --- |
| Title, Purpose | Text input and text area. |
| Line items | One row to start. Each row: Category (select), Description, Expense date (date picker, `YYYY-MM-DD`), Amount (text, 2 decimals), Receipt reference, **Remove**. |
| **Add line item** | Adds a row. Disabled at 20 rows. **Remove** is disabled when only one row is left. |
| Running total | Sum of valid amounts, shown as `฿1,250.00`. Display only; never sent. |
| **Submit** | Runs the shape checks, then sends. Disabled with a spinner while sending. |

### 2.2 States

| State | Behavior |
| --- | --- |
| Shape error (client) | Message under the field (§1.4). No request. |
| 422 / 400 from server | Each `errors[]` entry shown under its field. Form values kept. |
| 403 | "You can't create claims." page with no form. |
| Success | Confirmation: "Claim CLM-2026-000123 submitted", a **Submitted** badge (blue), the total, and links **View my claims** and **New claim**. |
| 5xx / network | Inline error with **Try again** and `requestId`. Form values kept. |

Finance Officers have no New Claim link (F-01). If they open `/claims/new` directly, the route guard checks `claim:create` and shows the 403 page.

## 3. API spec

### 3.1 Schemas

**`ClaimInput`** (request):

```json
{
  "title": "Bangkok workshop travel",
  "purpose": "Train fares and hotel for the 3 Nov workshop",
  "items": [
    {
      "category": "TRAVEL",
      "description": "Train BKK–Ayutthaya return",
      "expenseDate": "2026-11-03",
      "amount": "250.00",
      "receiptRef": "SRT-88412"
    }
  ]
}
```

| Field | Shape |
| --- | --- |
| `title` | string, 1–120 |
| `purpose` | string, 1–1000 |
| `items` | array of `LineItemInput` (count checked by BR-05, not the schema) |
| `items[].category` | enum (§1.3) |
| `items[].description` | string, 1–200 |
| `items[].expenseDate` | string, `format: date` |
| `items[].amount` | `Money`: string, pattern `^\d{1,6}\.\d{2}$` |
| `items[].receiptRef` | string, 1–50 |

Unknown fields, including `total`, are ignored (BR-08).

**`Claim`** (response, also used by F-03 and F-04):

```json
{
  "id": "0192f1c4-...",
  "claimNo": "CLM-2026-000123",
  "status": "SUBMITTED",
  "title": "Bangkok workshop travel",
  "purpose": "Train fares and hotel for the 3 Nov workshop",
  "total": "250.00",
  "returnCount": 0,
  "version": 0,
  "requester": {
    "id": "...",
    "name": "Anan K.",
    "position": "Staff, Section A"
  },
  "assignedReviewer": {
    "id": "...",
    "name": "Malee S.",
    "position": "Section Head A"
  },
  "firstSubmittedAt": "2026-11-05T03:00:00Z",
  "submittedAt": "2026-11-05T03:00:00Z",
  "decidedAt": null,
  "createdAt": "2026-11-05T03:00:00Z",
  "updatedAt": "2026-11-05T03:00:00Z",
  "items": [
    {
      "lineNo": 1,
      "category": "TRAVEL",
      "description": "Train BKK–Ayutthaya return",
      "expenseDate": "2026-11-03",
      "amount": "250.00",
      "receiptRef": "SRT-88412"
    }
  ],
  "history": [
    {
      "id": 1,
      "action": "SUBMIT",
      "fromStatus": null,
      "toStatus": "SUBMITTED",
      "actor": {
        "id": "...",
        "name": "Anan K.",
        "position": "Staff, Section A"
      },
      "comment": null,
      "createdAt": "2026-11-05T03:00:00Z"
    }
  ],
  "allowedActions": []
}
```

### 3.2 Operation

| Method | Path | Request | Success | Errors | Auth |
| --- | --- | --- | --- | --- | --- |
| POST | `/claims` | `ClaimInput` | 201 `{ "data": Claim }`, `ETag: "0"`, `Location: /api/v1/claims/{id}` | 400 `BAD_REQUEST`, 401, 403 `FORBIDDEN`, 422 `VALIDATION_FAILED` | `claim:create` |

A 422 lists **every** broken rule, one `errors[]` entry per field, each with `field`, `rule`, and `message` from §1.4.

## 4. Data spec

### 4.1 Migration `00002_claims.sql`

| Object | Detail (SDS §2.2) |
| --- | --- |
| Enums | `claim_status` (all 7 values), `claim_action` (all 8 values), `expense_category` (6 values). All values are added now, so later sprints need no enum migration. |
| Sequence | `claim_no_seq`, never reset. |
| `claims` | All SDS columns and CHECKs: `total_satang > 0 AND <= 5000000`; `return_count BETWEEN 0 AND 3`; `assigned_reviewer_id <> requester_id`; `status = 'DRAFT' OR assigned_reviewer_id IS NOT NULL`. Indexes `(requester_id, status)`, `(assigned_reviewer_id, status)`. |
| `line_items` | FK `claim_id` ON DELETE CASCADE; UNIQUE (`claim_id`, `line_no`); `line_no BETWEEN 1 AND 20`; `amount_satang > 0 AND <= 2000000`; `category <> 'MEALS' OR amount_satang <= 150000`. |
| `claim_history` | `id` identity; FK `claim_id` with no cascade; FK `actor_id`. |
| Trigger | `BEFORE UPDATE OR DELETE ON claim_history` raises an error (BR-11). |

### 4.2 Code

- `internal/money`: `Satang` type with JSON as a 2-decimal string. sqlc maps `*_satang` columns to it.
- Claim number: `CLM-<year of creation in Asia/Bangkok>-<nextval, 6 digits>`.
- `line_no` follows the order of `items` in the request, starting at 1.

## 5. Acceptance criteria

| ID | Requirement | Acceptance criterion |
| --- | --- | --- |
| F02-AC-01 | FR-02, FR-03, FR-07, BR-17 | `req.anan` submits a valid claim with 2 line items → 201. The claim has status `SUBMITTED`, a claim number matching `CLM-<Bangkok year>-<6 digits>`, `total` equal to the sum of the items, `assignedReviewer` = `rev.malee`, `firstSubmittedAt` = `submittedAt` = now, `version` 0, and one history entry (`SUBMIT`, from `null` to `SUBMITTED`, actor `req.anan`). |
| F02-AC-02 | FR-02, NFR-06 | The claim, its line items (in request order), and its history entry are stored in one transaction. If the history insert fails, no claim or line item row exists afterwards. |
| F02-AC-03 | BR-05 | 0 and 21 line items → 422 with `field = "items"` and `rule = "BR-05"`. 1 and 20 line items → 201. |
| F02-AC-04 | BR-05 | Amounts `"0.00"` and `"20000.01"` → 422 on `items[0].amount` with BR-05. `"0.01"` and `"20000.00"` → 201. `"1.005"` → 400 on `items[0].amount`. |
| F02-AC-05 | BR-06 | A total of 50,000.00 → 201. A total of 50,000.01 → 422 with `field = "total"` and BR-06. |
| F02-AC-06 | BR-12 | A `MEALS` item of 1,500.00 → 201; 1,500.01 → 422 with BR-12; a `TRAVEL` item of 1,500.01 → 201. Category `"FOOD"` → 400 on `items[0].category`. |
| F02-AC-07 | BR-07 | With the clock at 5 Nov 2026 10:00 Bangkok: expense date 6 Nov → 422 "Expense date cannot be in the future."; 5 Nov → 201; 7 Aug (90 days) → 201; 6 Aug (91 days) → 422 with the window message. With the clock at 6 Nov 00:30 Bangkok (5 Nov UTC), 6 Nov → 201. |
| F02-AC-08 | BR-08, FR-03 | A request with `"total": "1.00"` and items that sum to 1,250.00 → 201 with `total = "1250.00"`. |
| F02-AC-09 | BR-15 | `fin.wanida` calls `POST /claims` → 403 and no claim is stored. `claim.Service.Create` for an actor whose `reviewer_id` is NULL → `ErrForbidden`. |
| F02-AC-10 | BR-17 | After a claim is submitted, changing the requester's `reviewer_id` does not change the claim's `assigned_reviewer_id`. |
| F02-AC-11 | NFR-08 | A request that breaks two rules on two fields → one 422 with two `errors[]` entries. Each entry has `field`, `rule`, and the exact `message` from §1.4. |
| F02-AC-12 | FR-02, NFR-08 | Missing `title`, a `title` over 120 characters, a `purpose` over 1,000, a `description` over 200, a `receiptRef` over 50, and a bad date format each → 400 with `errors[].field` naming the field. |
| F02-AC-13 | FR-02, NFR-08 | On the New Claim form: rows can be added up to 20 and removed down to 1; the running total shows `฿` with 2 decimals; shape errors show under the field with no request; server `errors[]` show under the matching field and the input is kept; **Submit** is disabled while sending; success shows the claim number and a Submitted badge. |
| F02-AC-14 | BR-15 | A Finance Officer who opens `/claims/new` sees "You can't create claims." and no form. |

## 6. Software Test Specification

Levels as in F-01 §6. Module tests call `claim.Service` against `pgtest` PostgreSQL with a fixed clock (`2026-11-05T10:00:00+07:00` unless stated).

| Test | AC | Level | Precondition / input | Action | Expected result |
| --- | --- | --- | --- | --- | --- |
| F02-TEST-01 | F02-AC-01 | Integration | `req.anan` logged in; 2 valid items (250.00 + 1,000.00) | `POST /claims` | 201; `ETag: "0"`; body matches F02-AC-01; database has 1 claim, 2 line items, 1 history row |
| F02-TEST-02 | F02-AC-01 | Unit | Clock `2026-12-31T17:30:00Z` (1 Jan 2027 Bangkok) | Build the claim number for `nextval = 123` | `CLM-2027-000123` |
| F02-TEST-03 | F02-AC-02 | Integration | Test-only trigger on `claim_history` that raises on INSERT | `Create` a valid claim | Error returned; `claims` and `line_items` have no new rows |
| F02-TEST-04 | F02-AC-03 | Integration (module) | 0, 1, 20, 21 valid items | `Create` | `ValidationError{items, BR-05}`, ok, ok, `ValidationError{items, BR-05}` |
| F02-TEST-05 | F02-AC-04 | Unit | JSON strings `"0.00"`, `"0.01"`, `"20000.00"`, `"20000.01"`, `"1.005"`, `"1.5"`, `"-1.00"`, `"abc"` | Decode into `money.Satang` | First four decode (0, 1, 2000000, 2000001); the rest fail |
| F02-TEST-06 | F02-AC-04 | Integration (module) | One item each: 0.00, 0.01, 20,000.00, 20,000.01 | `Create` | BR-05 error, ok, ok, BR-05 error on `items[0].amount` |
| F02-TEST-07 | F02-AC-04 | Integration | Item amount `"1.005"` | `POST /claims` | 400 `BAD_REQUEST`; `errors[0].field = "items[0].amount"` |
| F02-TEST-08 | F02-AC-05 | Integration (module) | Items summing 50,000.00; items summing 50,000.01 | `Create` | ok; `ValidationError{total, BR-06}` |
| F02-TEST-09 | F02-AC-06 | Integration (module) | `MEALS` 1,500.00; `MEALS` 1,500.01; `TRAVEL` 1,500.01 | `Create` | ok; BR-12 on `items[0].amount`; ok |
| F02-TEST-10 | F02-AC-06 | Integration | Category `"FOOD"` | `POST /claims` | 400; `errors[0].field = "items[0].category"` |
| F02-TEST-11 | F02-AC-07 | Unit | Date rule with fixed clocks; dates 2026-11-06, 2026-11-05, 2026-08-07, 2026-08-06; midnight clock with 2026-11-06 | Check BR-07 | Future error, ok, ok, window error; midnight case ok |
| F02-TEST-12 | F02-AC-07 | Integration | Expense date 2026-11-06 | `POST /claims` | 422 with `items[0].expenseDate`, BR-07, "Expense date cannot be in the future." |
| F02-TEST-13 | F02-AC-08 | Integration | Body with `"total": "1.00"`; items 250.00 + 1,000.00 | `POST /claims` | 201; `total = "1250.00"`; `total_satang = 125000` |
| F02-TEST-14 | F02-AC-09 | Integration | `fin.wanida` logged in | `POST /claims` with a valid body | 403 `FORBIDDEN`; claim count unchanged |
| F02-TEST-15 | F02-AC-09 | Integration (module) | Test user with role `requester` and `reviewer_id` NULL | `Create` | `ErrForbidden`; no rows stored |
| F02-TEST-16 | F02-AC-10 | Integration (module) | Claim submitted by `req.anan` | Set `req.anan.reviewer_id = rev.somchai`; `Get` the claim | `assignedReviewer` is still `rev.malee` |
| F02-TEST-17 | F02-AC-11 | Integration | Item 1 date in the future; item 2 `MEALS` 1,600.00 | `POST /claims` | 422 with exactly two entries: `items[0].expenseDate` BR-07 and `items[1].amount` BR-12, messages from §1.4 |
| F02-TEST-18 | F02-AC-12 | Integration | One request per shape violation in F02-AC-12 | `POST /claims` | 400 for each; `errors[].field` names the field |
| F02-TEST-19 | F02-AC-13 | Component | MSW `POST /claims` returns 201, then 422 | Add rows to 20, remove to 1, type amounts, submit bad format, submit, get 422 | Add disabled at 20, Remove disabled at 1; total `฿1,250.00`; client message with no request; server messages under fields with values kept; button disabled while pending; confirmation with claim number |
| F02-TEST-20 | F02-AC-14 | Component | MSW `/auth/me` as `fin.wanida` | Open `/claims/new` | "You can't create claims." and no form |
| F02-TEST-21 | F02-AC-01, F02-AC-13 | E2E | Seeded database | Log in as `req.anan`, fill the form with 2 items, submit | Confirmation shows `CLM-…` and Submitted |

## 7. Feature Definition of Done

The Product DoD applies. For F-02 the evidence is:

- [ ] All F02-TEST rows pass in CI on the merge commit.
- [ ] TDD evidence for at least one rule (BR-05, BR-07, or BR-12): commit IDs of the failing test and the passing implementation, for #4.
- [ ] Migration `00002_claims.sql` runs up and down on an empty database.
- [ ] `docs/traceability.md` lists the actual test names and results for F-02 rows.

## 8. Revision history

| Version | Date | Change | Author |
| --- | --- | --- | --- |
| 0.1 | 2026-11-05 | Initial contract | 1005, 1045 |
| 0.2 | 2026-11-06 | Review fix (PR #23): workflow shows session, then contract validation, then route permission (SDS §1.3) | 1005, 1045 |
