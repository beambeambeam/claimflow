# Lab 6 Parts B–D – UAT results, triage, fixes, and retest

The record of Lab 6 after the handoff. The handout is [`lab-6-sprint-1-uat.md`](../../project-specifications/lab-6-sprint-1-uat.md) §3.2 to §3.4. Part A, the cases and the handoff for ClaimFlow, is [`uat-handoff.md`](uat-handoff.md).

ClaimFlow plays two roles:

- **Product Team** for ClaimFlow. The Queue team tests it (§3, §4, §5).
- **UAT Team** for the Queue and Service Management System (§2).

Student names and IDs are in the PDF only. This document uses GitHub logins.

## 1. Versions

| Item | Commit | Cloud Run revision | Notes |
| --- | --- | --- | --- |
| Sprint 1 baseline | `v0.0.3.0` = `ac5bce5` | – | Sprint 1 release (#69) |
| Tested version | `eed7b27` | `claimflow-00007-5vx` | Packaging and dependency changes only since `v0.0.3.0` ([handoff §1.2](uat-handoff.md#12-product-and-version)). Both findings were made on it |
| Defective code | `eed7b27` (same code as `v0.0.3.0`) | – | #86: `claimSchema` in `claim-form.tsx` and the `ClaimInput` / `LineItemInput` schemas in `openapi.yaml`. #87: `ClaimForm` submit handler |
| Lab 6 staging base | `f684459` | – | Handoff merged into `lab-6` (#85). The red runs use it as base |
| Fix #86 | `ecb4b2e`, `2054923`, `3da036a` → merge `6474b95` | – | PR #92 into `lab6-staging` |
| Fix #87 | `f97dec4`, `34e7198`, `496b22f` → merge `f498f49` | – | PR #91 into `lab6-staging` |
| Release PR head | `0f2dd01` | – | `lab-6` with `main` merged in (#88, #89) |
| Corrected `main` release | `bb5a0d1` | `claimflow-00010-985` | PR #93, merge commit, deployed 2026-11-21 16:29 UTC. Retested on this version |

Merge commits: `0f2dd01` merges `main` into `lab-6`, so the release also carries #88 (agent skills, no file under `apps`) and #89 (Bun dependency update). The reviewer of #93 checked that the release diff equals the reviewed result of #85, #91, and #92 for those files. The releases that deployed during the UAT window are in [handoff §4](uat-handoff.md#4-releases-during-the-uat-window).

No release tag: semantic-release found "no relevant changes" for `bb5a0d1` (CI run 44924470476), so the latest tag stays `v0.0.4.0`. The fix release is identified by its commit, which `/api/v1/health` returns.

## 2. Part B – ClaimFlow tests the Queue team's Sprint 1

| Item | Value |
| --- | --- |
| Product | Queue and Service Management System, <https://github.com/MacOverlorD/Queue_and_Service_Management_System_37-39> |
| Product Team | `MacOverlorD`, `TauForge` |
| UAT Team | ClaimFlow, `beambeambeam`, `Kiatisakk` |
| URL | <https://qms-6ba6hboj4a-as.a.run.app> |
| Tested version | `main` @ `e754975`, Cloud Run revision `qms-00006-z7c` |
| Cases | UAT-01 to UAT-04 from the Queue team's `dev_handoff.md` (shared privately) |
| Run | 2026-11-20 12:45:09–12:46:26 (UTC+7), each case once, in order UAT-01 to UAT-04 |
| Method | The handoff steps, followed in Chromium on the deployed app, with one browser window per user. Extra security and contract checks with `curl` |
| Record | Results and decision: [Queue #59](https://github.com/MacOverlorD/Queue_and_Service_Management_System_37-39/issues/59) |

Roles: `beambeambeam` ran the cases, wrote the reports #55–#58, and recorded the decision in #59. `Kiatisakk` solved the problem with the production environment and re-ran the cases.

### 2.1 Results

| Case | Time (UTC+7) | Actual result | Result |
| --- | --- | --- | --- |
| UAT-01 | 12:45:12–12:45:56 | Customer Three joined Finance Desk and got #2 WAITING, 1 person ahead. Operator One saw only Finance Desk and Registrar Desk. Call next, Start, and Complete moved #1 through CALLED and SERVING and off the panel. Without a reload, the customer window showed 0 ahead after 2.8 s, CALLED after 9.9 s, and dropped the ticket after 9.4 s | PASS |
| UAT-02 | 12:45:56–12:46:03 | Empty queue: "Nobody is waiting in this queue." Second join: "You already have an active ticket in this queue." and no new number. Closed queue: join button disabled. Call next while busy: "Finish the current customer before calling the next one." The next join after the refused one got #4, so the refusal used no number | PASS |
| UAT-03 | 12:46:03–12:46:17 | Signed out: 401 `UNAUTHENTICATED`. Customer Two on another user's ticket: 404 `NOT_FOUND`; operator page: "Not available"; operator API: 403 `FORBIDDEN_ROLE`. Operator Two on Registrar Desk: 403 `FORBIDDEN_QUEUE`. Each rejection body held only `{error:{code,message}}` | PASS |
| UAT-04 | 12:46:17–12:46:26 | Wrong password: "Invalid email or password." Registrar #5 WAITING stayed after F5 and after a new login, with its history. Logout ended the session on the server (the old cookie got 401). Operator data was correct after reload, but the selected queue reset to the first queue (#55) | PASS, with finding #55 |

End state: Finance Desk free, 0 waiting, last number 4. Registrar Desk and Student Affairs were not changed.

UAT-01 step 3 lists the queue selector as "Registrar Desk, Finance Desk". The app shows them sorted by name, as the Queue team's F-06 contract says. We judged the content, not the order.

Extra checks that passed: the `sid` cookie is `HttpOnly`, `Secure`, `SameSite=Lax`; a wrong password and an unknown email get the same reply; the dev seed password is refused on production; malformed login bodies get 4xx, never 500; a forged cookie gets 401; write endpoints refuse signed-out users (401), customers (403), and operators of another queue (403 `FORBIDDEN_QUEUE`); the public display API holds no names or emails; no sideways scrolling at 390 px on `/queues`, `/my-ticket`, and `/operator`.

### 2.2 Outgoing reports

| Issue | Finding | Severity | Product Team triage | Final state |
| --- | --- | --- | --- | --- |
| [#55](https://github.com/MacOverlorD/Queue_and_Service_Management_System_37-39/issues/55) | Operator console loses the selected queue on reload (UAT-04 step 8) | Minor | Accepted defect, not fixed in Sprint 1 | Accepted by us as a known limitation |
| [#56](https://github.com/MacOverlorD/Queue_and_Service_Management_System_37-39/issues/56) | No security response headers; `X-Powered-By: Express` sent | Minor | Enhancement: no Sprint 1 requirement covers response headers | Agreed: outside Sprint 1 scope |
| [#57](https://github.com/MacOverlorD/Queue_and_Service_Management_System_37-39/issues/57) | Production CORS allows `http://localhost:5173` with credentials | Minor | Accepted defect, not fixed in Sprint 1 | Accepted by us as a known limitation |
| [#58](https://github.com/MacOverlorD/Queue_and_Service_Management_System_37-39/issues/58) | Unknown `/api/*` routes return HTML, not the JSON error format | Minor | Accepted defect, fixed (Queue PR #60) | Retest PASS |

Each issue has the version, steps, expected and actual result, and evidence (screenshots or `curl` output).

### 2.3 Retest and decision

Retest of #58 by `beambeambeam`, 2026-11-21 20:09 (UTC+7), on `253b80e` (Cloud Run `qms-00007-zqx`): `GET /api/does-not-exist` returned 404, `Content-Type: application/json; charset=utf-8`, body `{"error":{"code":"NOT_FOUND","message":"Not found."}}`. PASS.

**Decision for the Queue team's Sprint 1: Accepted with documented minor limitations** (#55 and #57), on `253b80e` / `qms-00007-zqx`, recorded in [Queue #59](https://github.com/MacOverlorD/Queue_and_Service_Management_System_37-39/issues/59#issuecomment-7063068926) on 2026-11-21. #56 is an enhancement outside Sprint 1.

## 3. Part C – findings received on ClaimFlow

The Queue team tested `eed7b27` (`claimflow-00007-5vx`) on 2026-11-20. Both findings came from their exploratory tests (X-02 to X-04), not from a failing handoff step. They filed them as GitHub Issues and posted the links in `#qa`.

| Issue | Reporter | Finding | Case and requirement | Classification | Severity | Owner | Status |
| --- | --- | --- | --- | --- | --- | --- | --- |
| [#86](https://github.com/beambeambeam/claimflow/issues/86) | `MacOverlorD` | A title, purpose, line-item description, or receipt reference of only spaces is accepted, and the claim is stored blank (CLM-2026-000007, 000008, 000013, 000014) | X-02, X-03; FR-02, NFR-08, F02-AC-12 | Accepted defect | Minor | `beambeambeam` | Fixed in `bb5a0d1` (PR #92); retest PASS; closed 2026-11-22 |
| [#87](https://github.com/beambeambeam/claimflow/issues/87) | `MacOverlorD` | A double-click on **Submit** can create two identical claims (CLM-2026-000015 and 000016, 1 of 3 runs) | X-04; FR-02, F02-AC-13 ("Submit is disabled while sending") | Accepted defect | Major | `beambeambeam` | Fixed in `bb5a0d1` (PR #91); retest PASS; closed 2026-11-22 |

No duplicates, not-defects, or enhancements were received.

Reasons:

- **#86 is a defect.** F-02 §1.4 gives "Enter a title (up to 120 characters)." for an empty title, and the same for the other three fields. Text of only spaces is empty to a user, and the stored claim cannot be read or edited in Sprint 1. The triage reproduced it on `f684459` in both layers: `POST /api/v1/claims` returned 201, because the contract had only `minLength: 1`, and the form checked only `min(1)`. Severity minor: no money or access is affected, and the reviewer can reject the claim.
- **#87 is a defect.** F02-AC-13 requires **Submit** to be disabled while sending. The triage reproduced it in a component test: two clicks in one tick sent two create requests, because Submit went busy only after `create.isPending` rendered. Severity major: the requester does not see the duplicate, cannot withdraw it in Sprint 1, and the same expense can be approved and paid twice. A major defect blocks acceptance, so it had to be fixed and retested.
- Server-side idempotency (`Idempotency-Key` on `POST /claims`) is not part of the #87 fix. SDS §7.2 puts double-submit prevention in the UI, and refusing identical content on the server would block legitimate identical claims. It would be a separate enhancement.
- Claims stored blank or twice during UAT (CLM-2026-000007, 000008, 000013 to 000016) stay in the database. Sprint 1 has no edit, delete, or withdraw.

## 4. Part D – fixes

Both fixes went through the same path: Issue, fix branch from `lab6-staging`, teammate review with requested changes, CI, merge into `lab6-staging`, release PR into `main`, deploy, smoke test, retest.

### 4.1 #86 – whitespace-only text (PR #92)

Branch `fix/reject-blank-claim-text`, author `beambeambeam`.

The contract adds a non-blank `pattern` to the four text fields. It lists the JavaScript `\s` set as `\uXXXX` escapes, because Go RE2 `\s` is ASCII only; kin-openapi rewrites `\uXXXX` to `\x{XXXX}`, so the API and the form refuse the same text. The API answers 400 naming the field, rule `pattern`, with its §1.4 message. The form checks the same set with `/\S/u` and the same messages, and sends no request.

```diff
 ClaimInput.title / purpose, LineItemInput.description / receiptRef
   type: string
   minLength: 1
   maxLength: <n>
+  pattern: *notBlank   # &notBlank '[^\t\n\v\f\r \u00A0\u1680\u2000-\u200A\u2028\u2029\u202F\u205F\u3000\uFEFF]'
```

```diff
 claimSchema (apps/web/src/features/claims/components/claim-form.tsx)
-  title: z.string().min(1, "Enter a title (up to 120 characters).")
+  title: z.string().regex(notBlank, "Enter a title (up to 120 characters).")
   …same for purpose, description, receiptRef
```

Regression tests (excerpt of `apps/api/claims_create_test.go › TestF02TEST18ShapeViolationsAre400`):

```go
{"title of spaces", body(`"   "`, `"Purpose"`, validItem), fieldError{"title", "pattern", "Enter a title (up to 120 characters)."}},
{"purpose of spaces", body(`"Title"`, `" \t "`, validItem), fieldError{"purpose", "pattern", "Enter a purpose (up to 1,000 characters)."}},
{"description of spaces", body(`"Title"`, `"Purpose"`, itemWith("description", `"   "`)), fieldError{"items[0].description", "pattern", "Enter a description (up to 200 characters)."}},
{"receiptRef of spaces", body(`"Title"`, `"Purpose"`, itemWith("receiptRef", `"   "`)), fieldError{"items[0].receiptRef", "pattern", "Enter the receipt reference (up to 50 characters)."}},
{"title of no-break spaces", body(`"\u00a0\u00a0"`, `"Purpose"`, validItem), fieldError{"title", "pattern", "Enter a title (up to 120 characters)."}},
```

Red and green:

| Test | Red | Green |
| --- | --- | --- |
| F02-TEST-18, four ASCII whitespace cases | Base `f684459`: `status = 201, want 400` (claim stored with title `"   "`) | `ecb4b2e`: PASS, 400, `field=<field> rule=pattern` |
| F02-TEST-18, U+00A0, U+3000, U+FEFF, U+2028 cases | `ecb4b2e` (`\S` pattern): FAIL, `status = 201` | `2054923`: PASS, 400, `rule=pattern` |
| F02-TEST-19 `refuses text fields of only spaces the same as empty ones and sends no request` | Base `f684459`: FAIL, no message shown | `ecb4b2e`: PASS, four messages, no request sent |

Commands, run in `apps/api` and `apps/web`: `bun run test:coverage` (API 100% coverage, 759/759 statements; web 131 tests passed), `bun run check` at the root, and in `apps/api` `bun run format`, `bun run lint:fix` (0 issues), `bun run vuln` (no vulnerabilities).

Review by `Kiatisakk`:

- 2026-11-21 06:41 UTC, **changes requested**: the server's `\S` matches U+00A0, U+3000, and U+FEFF, but the form's `/\S/u` does not, so a title of two no-break spaces was refused by the form but stored by the API. Also: `docs/traceability.md` was not updated, and revision row 0.5 named the wrong author.
- 2026-11-21 08:01 UTC, **approved** after `2054923` and `3da036a`: the pattern and `/\S/u` agree on U+00A0 ×2, U+3000, U+FEFF, U+2028, `" \t "`, U+200B, `Title`, and `" x "`; F02-TEST-18 has the Unicode rows; the traceability rows record red and green.

CI on `3da036a`: `code_quality`, `test`, `api`, `build`, `image` passed. Merged into `lab6-staging` as `6474b95`.

### 4.2 #87 – double-click on Submit (PR #91)

Branch `fix/prevent-double-submit`, author `beambeambeam`.

`submit` returns early while `form.state.isSubmitting` is true. TanStack Form sets it before its first `await`, so it holds for a second click in the same tick. `onSubmit` awaits `create.mutateAsync`, so `isSubmitting` stays true until the request settles, and the Submit spinner follows it. No API change.

```diff
   const submit = () => {
+    if (form.state.isSubmitting) {
+      return;
+    }
     form.setErrorMap({ onSubmit: { fields: {} } });
     void form.handleSubmit();
   };
-  onSubmit: ({ value }) => { create.mutate({ body: claimSchema.parse(value) }, { onError }) }
+  onSubmit: async ({ value }) => { try { await create.mutateAsync({ body: claimSchema.parse(value) }) } catch (error) { … } }
-  <Button isLoading={create.isPending} label="Submit" />
+  <form.Subscribe selector={(state) => state.isSubmitting}>
+    {(isSubmitting) => <Button isLoading={isSubmitting} label="Submit" />}
```

Regression test: `apps/web/src/features/claims/__test__/claim-form.test.tsx › F02-TEST-19 (F02-AC-13): New Claim form › sends one request when Submit is double-clicked`. It clicks Submit twice in one `act`, the way a fast double-click arrives before React renders, delays the 201 by 300 ms, asserts `aria-busy="true"` on Submit, and asserts exactly one create request.

| Run | Result |
| --- | --- |
| Red: the new test against the base `claim-form.tsx` at `f684459` | FAIL: `AssertionError: expected [ {…}, {…} ] to have a length of 1 but got 2` |
| Green: `f97dec4` | PASS, one request sent; the claim-form file 11/11 |
| After merging `lab6-staging` (#92) into the branch, `496b22f` | Claim-form tests 13/13 |

The test and the fix are one commit, so there is no red commit. The reviewer ran the red case separately by swapping in the base file.

Commands: `bun run check` at the root (pass) and `bun run test:coverage` in `apps/web` (131 tests passed in 16 files; 1 of 7 runs had one unrelated timeout, the next 6 passed). The F02-TEST-19 `describe` now has a 30 s timeout.

Review by `Kiatisakk`:

- 2026-11-21 06:41 UTC, **changes requested**: the fix and test are correct (red on base, green on head, `isSubmitting` set at `FormApi.js:519` before the first `await`), but `docs/traceability.md` must name the new case and its red evidence. Non-blocking: assert the busy state, and raise the file's timeout.
- 2026-11-21 08:18 UTC, **approved** after `34e7198`: traceability updated, `aria-busy` asserted, one 30 s timeout on the `describe`.

CI on `34e7198` and `496b22f`: `code_quality`, `test`, `api`, `build`, `image` passed. Merged into `lab6-staging` as `f498f49`.

### 4.3 Release and deployment (PR #93)

| Step | Evidence |
| --- | --- |
| Release PR | #93 `chore: release Lab 6`, `lab-6` into `main`, author `beambeambeam` |
| Review | `Kiatisakk` approved 2026-11-21 13:27 UTC: `git diff origin/main origin/lab-6` covers 11 files and equals the reviewed result of #85, #91, and #92; `openapi.yaml` has the final Unicode-aware pattern; no migration |
| CI on the PR head `0f2dd01` | Run 44905878321: `code_quality`, `test`, `api`, `build`, `image` passed |
| Merge | Merge commit `bb5a0d1`, 2026-11-21 16:25 UTC. The merge redeploys the demo, so the UAT Team is told in `#qa` first ([handoff §1.2](uat-handoff.md#12-product-and-version)) |
| CI on `main` | Run 44924470476: all jobs passed; `release` made no tag (§1) |
| Deploy | Deploy demo run 44924677973: revision `claimflow-00010-985` serving 100% of traffic at 16:29 UTC; smoke test passed |
| Version check | `GET /api/v1/health` returns `{"status":"ok","version":"bb5a0d1a6c3db99d69d6187b09b4d66fd0be01c5"}` |

## 5. Independent retest and Sprint 1 decision

Retests by `MacOverlorD` for the Queue team, on `bb5a0d1` (read from `/api/v1/health`), 2026-11-22 (UTC+7), as `req.niran`:

| Issue | Time (UTC+7) | Actual result | Result |
| --- | --- | --- | --- |
| #86 | 03:15–03:21 | New Claim with one valid item, one field at a time set to 3 spaces. Title: "Enter a title (up to 120 characters)."; Purpose: "Enter a purpose (up to 1,000 characters)."; Description: "Enter a description (up to 200 characters)."; Receipt reference: "Enter the receipt reference (up to 50 characters)." No claim created. A direct `POST /api/v1/claims` with each field of 3 spaces returned 400 with the matching field error | PASS ([comment](https://github.com/beambeambeam/claimflow/issues/86#issuecomment-7068894155)) |
| #87 | 03:21–03:25 | Runs 1–3, double-click on Submit: one claim each (CLM-2026-000019, 000020, 000021). Run 4, three `click()` calls in one tick: one claim (000022). My Claims shows 000019 to 000022, consecutive, no duplicates. Regression: `rev.somchai` approved CLM-2026-000019 normally at 03:34 | PASS ([comment](https://github.com/beambeambeam/claimflow/issues/87#issuecomment-7068894510)) |

The Queue team's evidence IDs are B-R86-01 to 04, B-R87-01, 04, 05, and B-RREG, in their Lab 6 report. Both comments end "OK to close from the UAT side."

After the retests, `MacOverlorD` posted the final decision on both Issues at 2026-11-22 11:27 UTC ([#86](https://github.com/beambeambeam/claimflow/issues/86#issuecomment-7078452563), [#87](https://github.com/beambeambeam/claimflow/issues/87#issuecomment-7078452879)): "Final Sprint 1 UAT decision: Accepted, on `bb5a0d1` / `claimflow-00010-985`, tester Lappawat Laohasoot, 2026-11-22."

**Decision for ClaimFlow's Sprint 1: Accepted**, on `bb5a0d1` / `claimflow-00010-985`. No finding is open.

## 6. Contributions

| Work | `beambeambeam` | `Kiatisakk` |
| --- | --- | --- |
| Part A: UAT cases and handoff (#85) | Author | Reviewer (changes requested, then approved) |
| Part B: tests of the Queue team, reports #55–#58, decision #59 | Run, reports, decision | Solved the production environment problem, re-ran the cases |
| Part C: triage of #86 and #87 | Triage briefs | – |
| Part D: fixes #92 and #91 | Author | Reviewer (changes requested, then approved, on each) |
| Part D: release #93 | Author, merge | Reviewer |

No pair-programming sessions; the PRs record "Pair work: None".

## 7. AI use

- The triage briefs on #86 and #87 were drafted with an AI agent and say so ("This was generated by AI during triage"). Each brief records how the finding was reproduced.
- An AI coding agent helped write the fixes, the tests, and these documents. Every change went through teammate review and CI.

## 8. Known limitations

- No release tag for `bb5a0d1` (§1).
- The claims stored blank or twice during UAT stay in the demo database. Sprint 1 cannot edit, delete, or withdraw them.
- The demo redeployed three times during the UAT window ([handoff §4](uat-handoff.md#4-releases-during-the-uat-window)). Only `bb5a0d1` changed Sprint 1 behavior.
- No server-side duplicate guard for `POST /claims`. A retry from a second tab can still create a second claim; an `Idempotency-Key` would be a later enhancement.
- The Playwright E2E specs run on a local stack, not in CI (#66).
