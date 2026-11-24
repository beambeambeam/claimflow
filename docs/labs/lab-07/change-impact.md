# Lab 7 Part A – Change impact and revised contract

The change request, its impact, and the contract revision for Lab 7, written before any code changes. The handout is [`lab-7-sprint-1-maintenance.md`](../../project-specifications/lab-7-sprint-1-maintenance.md) §3.1. The change request is issue [#99](https://github.com/beambeambeam/claimflow/issues/99) (`L7/01`). The revised contract is the [SRS](../../srs.md), the [F-04 contract](../../sprints/sprint-1/F-04-review-decision.md), the [System SDS](../../system-sds.md), and the [traceability matrix](../../traceability.md).

Student names and IDs are in the PDF only. This document uses GitHub logins.

## 1. Change request

| Item | Value |
| --- | --- |
| Issue | [#99](https://github.com/beambeambeam/claimflow/issues/99) `L7/01: Raise the minimum rejection reason length from 5 to 10 characters (BR-04)`, label `lab7` |
| Type | Changed requirement, not a defect. The 5-character minimum matches the accepted Sprint 1 contract |
| Original requirement | FR-06 (a rejection requires a reason) and BR-04 (reason required, 5–500 characters). Implemented in F-04 Review decision, Sprint 1 |
| Old rule | A reject reason is required: 5–500 characters after trimming, counted in Unicode characters |
| New rule | A reject reason is required: 10–500 characters after trimming, counted in Unicode characters. The SRS minimum is 10 for every BR-04 reason (reject, return, final reject); only reject is implemented today |
| Reason | A reason such as "No pic" (6 characters, accepted today) gives the requester nothing to act on. A longer minimum makes the reviewer explain the decision |
| Who may reject | Unchanged: only the claim's assigned reviewer, never its requester (BR-01, BR-02) |
| Out of scope | The approve comment rule; return (F-06, Sprint 2) and final reject (F-08, Sprint 3); the database schema |

### 1.0 Scope of the change

The change touches the API rule and its messages (`rules.go`), the OpenAPI contract and the generated code, the web decision dialog (a client-side minimum that disables Reject and shows the field message), the SRS, F-04 contract, System SDS and STS, four test files, the E2E spec, and the deploy. At its core it is a one-number rule change (5 to 10) with a UI behavior added.

### 1.1 Old and new behavior

`POST /claims/{id}/reject` by the assigned reviewer on a `SUBMITTED` claim with the current `If-Match`:

| Reason (after trimming) | Old result | New result |
| --- | --- | --- |
| Missing, empty, or blank | 422 `comment`, BR-04, "Enter a reason (5 to 500 characters)." | 422 `comment`, BR-04, "Enter a reason (10 to 500 characters)." |
| 1–4 characters | 422 `comment`, BR-04, "Reason must be at least 5 characters." | 422 `comment`, BR-04, "Reason must be at least 10 characters." |
| 5–9 characters | 200, `REJECTED` | 422 `comment`, BR-04, "Reason must be at least 10 characters." |
| 10 characters | 200, `REJECTED` | 200, `REJECTED` |
| 500 characters | 200, `REJECTED` | 200, `REJECTED` (unchanged) |
| 501 characters | 422, "Comment cannot be more than 500 characters." | Unchanged |
| Approve comment: none, or up to 500 / 501 characters | 200 / 422 | Unchanged |

After any 422, the claim's status, `version`, and history are unchanged, as before. 401, 403, 404, 409, and 428 keep their current conditions and precedence (F-04 §1.4): the comment rule is step 6, after the access and status checks.

## 2. Versions

| Item | Commit | Cloud Run revision | Notes |
| --- | --- | --- | --- |
| Accepted Sprint 1 baseline | `v0.0.3.0` = `ac5bce5` | – | Sprint 1 release (#69). 5–500 rule |
| Accepted Lab 6 release | `bb5a0d1` | `claimflow-00010-985` | PR #93, the Sprint 1 release after the UAT fixes ([Lab 6 report §1](../lab-06/uat-report.md#1-versions)). 5–500 rule |
| Pre-change implementation | `214d1d5` (`main` and `lab-7`) | `claimflow-00015-vh7` | The commit the red run uses. Deployed by Deploy demo run 45040335622 on 2026-11-22 15:49 UTC (the run log names the image tag and the revision). 5–500 rule |
| Contract revision | `78fe104` | – | This branch, `docs/lab-7-reject-reason-contract`, 2026-11-24. SRS 0.3, F-04 0.8, SDS 0.18, traceability. Docs only |
| Change-impact document | The commit that adds this file | – | Docs only |
| Instructor/TA approval | Not required, per the course announcement "Lab 7 Spec Sheet Change" |
| Teammate review of the contract | TBD | – | PR into `lab-7`, reviewed by `beambeambeam` before any code change |
| Red test commit | TBD | – | Part B |
| Green implementation commit | TBD | – | Part B |
| Accepted `main` release | TBD | TBD | Part B |

Intervening development between `bb5a0d1` and `214d1d5`: four docs-only PRs (#94, #95, #96, #98) and #97 (`fbf4382`, Sprint 1 follow-ups), which changed 43 files under `apps`. #97 touched files in this change's scope (`decision-panel.tsx`, `contract.ts`, `zod.gen.ts`, `decision-panel.test.tsx`): the counter now reads its limit from the contract, and a 403 on a decision opens the access page. It did not change BR-04: `MinReasonLength = 5` and both messages are the same in `ac5bce5`, `bb5a0d1`, and `214d1d5` (`apps/api/internal/claim/rules.go`). Check with `git diff bb5a0d1 214d1d5 -- apps/api/internal/claim/rules.go`.

## 3. Impact

### 3.1 Trace

| Requirement | Feature | SRS | Feature SDS (F-04) | System SDS | AC | STS tests | Code modules |
| --- | --- | --- | --- | --- | --- | --- | --- |
| BR-04 reject reason | F-04 (S1) | §7 BR-04 **revised** | §1.5 **revised**; §1.6 boundary row **revised**; §2.2 Reject row **revised**; §1.4 step 6 and §3 unchanged | §2.2 `claim_history.comment` **revised**; §3.3, §4.2 unchanged | F04-AC-06, F04-AC-18 **revised**; F04-AC-05 unchanged | F04-TEST-06, 07, 21 **revised**; F04-TEST-05, 24 unchanged | `apps/api/internal/claim/rules.go` (`MinReasonLength`, `checkComment`); `apps/api/openapi.yaml` descriptions and the generated files (§3.2) |
| NFR-08 field messages naming the rule | F-04 (S1) | §8 NFR-08 unchanged | §1.5 messages **revised**; §2.2 | §4.2 unchanged | F04-AC-06, F04-AC-18 **revised** | F04-TEST-06, 07, 21 **revised**; F04-TEST-24 unchanged | `rules.go` messages; `apps/web/src/features/reviews/components/decision-panel.tsx` (minimum and field message, Part B) |
| FR-06 reject with a reason | F-04 (S1) | §6 FR-06 unchanged | §1.3 unchanged | §3.2 unchanged | F04-AC-05 unchanged | F04-TEST-05, 24 unchanged | `apps/api/internal/claim/service.go` `Act`, `handler.go` `RejectClaim`: unchanged |
| BR-04 return, final reject | F-06 (S2), F-08 (S3) | §7 BR-04 **revised** | Not in Sprint 1. The F-06 draft in PR #100 uses `MinReasonLength`, not a literal, and names L7/01 | – | F06-AC-02 (planned, PR #100) | F06-TEST-02, 03 (planned) | Inherit `MinReasonLength` when built |

### 3.2 Places that state the old rule

Part A revises the documents. Part B changes the rest after the review, in this order: tests (red), then code and the OpenAPI descriptions, then `bun run generate`. Every line below was checked on `214d1d5`.

| Place | Lines | States | Part |
| --- | --- | --- | --- |
| `docs/srs.md` BR-04 | 151 | 5–500 | A, done |
| `docs/sprints/sprint-1/F-04-review-decision.md` §1.5, §1.6, F04-AC-06, F04-TEST-06, 07 | 76, 81, 82, 89, 202, 229, 230 | 5–500, both messages, 4/5 boundary | A, done |
| `docs/system-sds.md` `claim_history.comment` | 285 | 5–500 | A, done |
| `docs/traceability.md` BR-04 and NFR-08 rows | 44, 72 | Results recorded under 5–500 | A, done (marked as results of the 5–500 rule; new results pending) |
| `apps/api/internal/claim/rules.go` | 61 (`MinReasonLength = 5`), 126 (comment), 138, 140 (messages) | Rule and messages | B |
| `apps/api/openapi.yaml` | 280 (`rejectClaim` description), 663 (`ClaimDecision.comment` description) | 5–500 | B |
| Generated from `openapi.yaml` | `apps/api/internal/api/api.gen.go` 288; `apps/web/src/lib/api/types.gen.ts` 126; `sdk.gen.ts` 174; `@tanstack/react-query.gen.ts` 272 | 5–500 | B, by `bun run generate` |
| `apps/api/internal/claim/act_test.go` (F04-TEST-06) | 31, 35, 36, and cases 48–52 | Messages, 4/5 boundary | B, red first |
| `apps/api/claims_review_test.go` (F04-TEST-07) | 470–473 | Messages, 4-character case | B, red first |
| `apps/web/src/features/reviews/__test__/decision-panel.test.tsx` (F04-TEST-21) | 232, 244 | Mocked 422 message; new 9/10-character step | B, red first |
| `apps/web/src/features/reviews/components/decision-panel.tsx`, `apps/web/src/lib/contract.ts` | – | No minimum in the dialog | B |
| `apps/web/e2e/f04-review-decision.spec.ts` (F04-TEST-24) | 50 | "Enter a reason (5 to 500 characters)." | B |

Not changed, with the reason:

- `apps/web/src/lib/api/zod.gen.ts` 134 and `x-max-length: 500` (`openapi.yaml` 664): they state only the upper bound. Part B adds the minimum to the contract (`x-min-length`, read like `x-max-length`) so the dialog reads both limits from one source; this is the one OpenAPI schema change beyond the descriptions.
- `docs/labs/lab-06/uat-handoff.md` 254, 265, 266 and `docs/reports/labs/06/content/part-a-handoff.tex` 561, 589, 590: the Lab 6 UAT-05 case as it ran on the 5–500 release. They are records of that run, so they stay as written.

### 3.3 Considered

| Concern | Applies | Reason |
| --- | --- | --- |
| Callers and dependencies | Yes, one path | `checkComment` (`rules.go` 129) has one caller, `checkAct` (`service.go` 260), used by `Act` for approve and reject. Approve uses only the upper bound, which does not change. `RejectClaim` (`handler.go` 160) passes the body through unchanged. F-06 return reuses the constant when it is built (PR #100) |
| API validation | Yes, descriptions and one extension | The schema sets no `minLength` on `comment` (F-04 §3), so the BR-04 check in the claim module decides every length case and the 400 path is not involved. The two descriptions in `openapi.yaml` change, and `comment` gains `x-min-length: 10` for the web (UI validation); the generated copies follow |
| UI validation | Yes | The Reject dialog gains a client-side minimum (decision on #99 AC 5): **Reject** is disabled while the trimmed reason has fewer than 10 characters and the field shows "Reason must be at least 10 characters." The limit is read from the contract like the counter's 500. The API still checks the rule and its 422 message still shows under the field (F-04 §2.2). Behavior for 10–500 characters is unchanged |
| Roles and ownership | No | The comment rule is step 6 of F-04 §1.4, after the visibility, relation (BR-01, BR-02), and status (BR-03) checks. Who may reject, and the 403 and 409 results, do not change |
| Persistence | No | `claim_history.comment` is `VARCHAR(500)` NULL with no length CHECK (`apps/api/migrations/00002_claims.sql` 64). The trimmed reason is stored as before |
| Historical data | No | `REJECT` rows with 5–9 character reasons stay as stored. History is append-only (BR-11, trigger tested by F04-TEST-18), and reads never check BR-04 again. The seed's reject reason, "Receipt is missing." (19 characters, `apps/api/seed_claims.go` 54), is valid under both rules |
| Migrations | No | No schema or data change, so no migration |
| Transactions and concurrency | No | The 422 is returned before the conditional `UPDATE`, as before, so nothing is written and BR-13 handling is unchanged |
| Configuration and deployment | No | The minimum is a constant in the claim module, not an environment variable. The release goes through the Lab 5 pipeline as usual |
| ADRs | No conflict | ADR-0001 (Finance Officer reviews the Department Head) is about who decides, not the reason length |

## 4. Behavior that must not change

Part B runs the full API suite (`bun run test:coverage` in `apps/api`), the full web suite, and the F-04 E2E spec on a local seeded stack. These tests protect the behavior below. Apart from the BR-04 boundary cases listed in §3.2, none of them is edited.

| Behavior | Rule | Regression tests |
| --- | --- | --- |
| Upper bound 500 on reject and approve; approve comment optional | BR-04 | F04-TEST-06 cases for 500 and 501 characters and the approve cases; F04-TEST-07 cases for 501 characters; `TestContractCommentLimitMatchesModule` (`apps/api/validate_test.go` 91) |
| Trimming and counting in Unicode characters | BR-04 | F04-TEST-06 cases with blanks around and Thai text (moved to the 9 and 10 boundary) |
| A refused decision changes nothing | BR-03, NFR-06 | F04-TEST-06 (state after the refused `Act`), F04-TEST-07 (`readState` before and after), F04-TEST-14 |
| A valid reject stores the reason and the requester sees it | FR-06, FR-07 | F04-TEST-05 ("Missing receipt", 15 characters), F04-TEST-24 ("Receipt missing"), F04-TEST-21 reject success case ("Receipt is missing.") |
| No self-approval | BR-01 | F04-TEST-08, 09, 10 |
| Only the assigned reviewer decides; no claim data in a 403 | BR-02, NFR-02 | F04-TEST-11, 12, 13 |
| Status and version checks run before the reason check | BR-03, BR-13 | F04-TEST-14; `TestDecidedClaimWithBlankReasonIsInvalidTransition` (`apps/api/claims_review_test.go` 551): a blank reason on a decided claim is 409 `INVALID_TRANSITION`, not 422 |
| Concurrent and stale decisions | BR-13 | F04-TEST-15, 16; `TestDecisionWithStaleVersionOnDecidedClaimIsConflict` (`claims_review_test.go` 534) |
| Status change and history in one transaction | NFR-06 | F04-TEST-17; `TestDecisionHistoryFailureIs500AndChangesNothing` (`claims_review_test.go` 563) |
| History is append-only; old reasons stay | BR-11 | F04-TEST-18; F04-TEST-22 (seed reject reason) |
| Review Queue, decision panel states | FR-05, NFR-08 | F04-TEST-01, 02, 03, 20, 21 (counter, 409, 403 cases) |
| Approve journey | FR-06, FR-07 | F04-TEST-04, 23 |

## 5. Revised contract

Each revised document carries a revision entry: SRS 0.3, F-04 contract 0.8, System SDS 0.18, and a 2026-11-24 row in the traceability change log.

### 5.1 SRS §7 BR-04 (0.2 → 0.3)

Old:

> **Reason required:** reject, return, and final reject need a non-blank comment of 5–500 characters. Approve may include an optional comment.

New:

> **Reason required:** reject, return, and final reject need a non-blank comment of 10–500 characters. Approve may include an optional comment.

### 5.2 F-04 §1.5 Comment rule (0.7 → 0.8)

Old:

> - `reject`: required, 5–500 characters.
>
> | Input | Message |
> | --- | --- |
> | Missing or blank (reject) | "Enter a reason (5 to 500 characters)." |
> | 1–4 characters (reject) | "Reason must be at least 5 characters." |
> | More than 500 (reject or approve) | "Comment cannot be more than 500 characters." |

New:

> - `reject`: required, 10–500 characters.
>
> | Input | Message |
> | --- | --- |
> | Missing or blank (reject) | "Enter a reason (10 to 500 characters)." |
> | 1–9 characters (reject) | "Reason must be at least 10 characters." |
> | More than 500 (reject or approve) | "Comment cannot be more than 500 characters." |

### 5.3 F-04 §1.6 boundary row

Old: `Reject reason of 4 / 5 / 500 / 501 characters` → `422 / 200 / 200 / 422`

New: `Reject reason of 9 / 10 / 500 / 501 characters` → `422 / 200 / 200 / 422`

### 5.4 F-04 §2.2 Reject control

Old:

> Opens a dialog with a required reason field, a counter (`0/500`), the field messages from §1.5, **Reject** and **Cancel**.

New:

> Opens a dialog with a required reason field, a counter (`0/500`), the field messages from §1.5, **Reject** and **Cancel**. **Reject** stays disabled while the trimmed reason has fewer than 10 characters, and once any text is entered the field shows "Reason must be at least 10 characters." The limits come from the API contract (`minLength` and `maxLength`), as the counter's does. The API still checks the rule: a 422 shows its §1.5 message under the field (row 422).

### 5.4a F04-AC-18 (decision panel)

Old:

> … Approve asks for confirmation; Reject needs a reason and shows the field message. …

New:

> … Approve asks for confirmation; Reject needs a reason of at least 10 characters: its button is disabled below that and the field shows the message, and a 422 shows the API's message. …

### 5.5 F04-AC-06

Old:

> Reject with a missing or blank reason, or 4 or 501 characters → 422 with `field = "comment"`, `rule = "BR-04"`, and the message from §1.5. 5 and 500 characters → 200. Approve with a 501-character comment → 422.

New:

> Reject with a missing or blank reason, or 9 or 501 characters → 422 with `field = "comment"`, `rule = "BR-04"`, and the message from §1.5. 10 and 500 characters → 200. Approve with a 501-character comment → 422.

### 5.6 STS: F04-TEST-06, F04-TEST-07, and F04-TEST-21

| Test | Level | Precondition | Old input / expected result | New input / expected result |
| --- | --- | --- | --- | --- |
| F04-TEST-06 | Integration (module) | `SUBMITTED` claim | `Act(REJECT)` with no comment, `"   "`, 4, 5, 500, 501 characters; `Act(APPROVE)` with 501 → `ValidationError{comment, BR-04}` ×3, ok, ok, BR-04 error; BR-04 error | `Act(REJECT)` with no comment, `"   "`, 9, 10, 500, 501 characters; `Act(APPROVE)` with 501 → same expected results |
| F04-TEST-07 | Integration | `SUBMITTED` claim | `POST …/reject` with no `comment`, `"   "`, a 4-character reason → 422 each, `comment`, BR-04, "Enter a reason (5 to 500 characters)." ×2, "Reason must be at least 5 characters." | `POST …/reject` with no `comment`, `"   "`, a 9-character reason → 422 each, `comment`, BR-04, "Enter a reason (10 to 500 characters)." ×2, "Reason must be at least 10 characters." |

F04-TEST-21 (component, covers F04-AC-18) adds one step to its Reject case: type a 9-character reason, then a 10-character one. Expected: **Reject** is disabled with "Reason must be at least 10 characters." at 9 characters and enabled at 10; the existing 422, success, 409, and 403 steps are unchanged. F04-TEST-06 and 07 cover F04-AC-06. F04-TEST-24 keeps its input, "Receipt missing" (15 characters).

### 5.7 System SDS §2.2 `claim_history.comment` (0.17 → 0.18)

Old:

> NULL. Required (5–500 non-blank chars) for REJECT, RETURN, and FINAL_REJECT (BR-04), checked by the claim module.

New:

> NULL. Required (10–500 non-blank chars) for REJECT, RETURN, and FINAL_REJECT (BR-04), checked by the claim module when the entry is written. Entries written under the earlier 5-character minimum are kept as stored and not checked again (BR-11).

### 5.8 Traceability

The BR-04 and NFR-08 rows keep their recorded passes and red/green commits (`2a2f76f`, `be4c545`, #33), now marked as results of the 5–500 rule, with the 10–500 results pending. The change log has a 2026-11-24 row for #99.

## 6. Review

| Item | Value |
| --- | --- |
| Teammate review of the contract | Reviewer `beambeambeam` · PR: TBD · Date: TBD |
| Rule | No code change starts before the review is recorded |
