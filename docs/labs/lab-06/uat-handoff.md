# Lab 6 Part A – Sprint 1 UAT cases and handoff

What the UAT Team needs to test the ClaimFlow Sprint 1 release without help from the Product Team. The handout is [`lab-6-sprint-1-uat.md`](../../project-specifications/lab-6-sprint-1-uat.md) §3.1. The Sprint 1 contract is the four feature contracts in [`docs/sprints/sprint-1`](../../sprints/sprint-1) (F-01 to F-04) and the [SRS](../../srs.md).

Items marked **TODO** are not confirmed yet. The Product Team fills them in before the handoff.

## 1. Handoff

### 1.1 Teams

| Role | Team | Repository | GitHub logins |
| --- | --- | --- | --- |
| Product Team | ClaimFlow | <https://github.com/beambeambeam/claimflow> | `Kiatisakk`, `beambeambeam` |
| UAT Team | Queue and Service Management System | <https://github.com/MacOverlorD/Queue_and_Service_Management_System_37-39> | `MacOverlorD`, `TauForge` |

Student names and IDs are in the PDF only. Coordination happens in the private Discord channel `#qa`. **TODO**: UAT session dates (first run, defect fixes, retest), before the 22 November 2026 deadline.

### 1.2 Product and version

| Item | Value |
| --- | --- |
| Product | ClaimFlow: staff submit expense claims, and the assigned reviewer approves or rejects them |
| Repository | <https://github.com/beambeambeam/claimflow> |
| Sprint 1 release | Tag `v0.0.3.0`, commit `ac5bce5a0c827f0ef9c6d0841b00f0d0c5839801` (`chore: release Sprint 1 (#69)`) |
| Deployed commit | `eed7b278b046450c894337b02341271fd0987c56` (head of `main`), read from `/api/v1/health` on 2026-11-19 |
| Cloud Run revision | `claimflow-00007-5vx`, serving 100% of traffic (Deploy demo run 44533229870, 2026-11-19 13:26 UTC) |
| URL | <https://claimflow-5bcxewbqkq-as.a.run.app> |
| Version check | Open <https://claimflow-5bcxewbqkq-as.a.run.app/api/v1/health>. It returns `{"status":"ok","version":"<commit SHA>"}` |

The deployed commit is newer than the Sprint 1 tag. Between `v0.0.3.0` and `eed7b27` the application changed only in packaging and dependencies: the API serves the web build from one image (`apps/api/web.go`, Lab 5), `APP_VERSION` was added to the config, and Go and Bun dependencies were updated. Nothing in `apps/api/internal` or `apps/web/src` changed, so the Sprint 1 behavior and the API contract (`openapi.yaml`) are the same as in the tag. Check with `git diff --stat v0.0.3.0 eed7b27 -- apps`.

Every merge into `main` deploys (Lab 5). The Product Team tells the UAT Team before any merge into `main` during the UAT window. Record the `version` from `/api/v1/health` at the start of each test session. If it changes during a session, stop, tell the Product Team, and record the new version.

### 1.3 Scope accepted for test

In scope: Sprint 1 features F-01 to F-04 ([feature inventory](../../features.md)).

| Feature | What the user can do |
| --- | --- |
| F-01 Authentication and session | Log in with a seeded account, see the name and role in the header, see only the links of the role, log out |
| F-02 Submit claim | Create a claim with 1–20 line items and submit it. The server checks the amount, total, Meals, and date rules, and calculates the total |
| F-03 My claims | List your own claims and open one: line items, total, status, assigned reviewer, history. Other members' claims are hidden |
| F-04 Review decision | A Reviewer or Finance Officer sees the `Submitted` claims assigned to them and approves (optional comment) or rejects (required reason) |

Not in scope (Sprint 2 and 3). Do not report these as defects:

- Save as draft, edit, delete (F-05), return and resubmit (F-06), withdraw (F-07). In Sprint 1 the owner has no actions on a submitted claim.
- Two-stage approval (F-08). In Sprint 1, an approval is final at any amount, so a 40,000.00 THB claim approved by a reviewer is `Approved`.
- Filter and search (F-09), decision summaries (F-10), Finance Queue.
- Sign-up, password change, and account management. Only seeded accounts exist.

### 1.4 Accounts

The seed creates seven synthetic accounts. All seven have the same password. The Product Team posts it in the private Discord channel `#qa`, which only the two teams can read (**TODO**: date posted). It is not in this document or in Git.

| Account label | Display name | Role (header label) | Position | Assigned reviewer | Used in |
| --- | --- | --- | --- | --- | --- |
| `req.anan` | Anan K. | Requester | Staff, Section A | `rev.malee` | UAT-01, 03, 06, 07 |
| `req.ploy` | Ploy | Requester | Staff, Section A | `rev.malee` | UAT-02, 03 |
| `req.niran` | Niran | Requester | Staff, Section B | `rev.somchai` | UAT-05 |
| `rev.malee` | Malee S. | Reviewer | Section Head A | `rev.prasert` | UAT-01, 03, 04, 06, 07 |
| `rev.somchai` | Somchai | Reviewer | Section Head B | `rev.prasert` | UAT-03, 05 |
| `rev.prasert` | Prasert | Reviewer | Department Head | `fin.wanida` | UAT-04 |
| `fin.wanida` | Wanida | Finance Officer | Finance Officer | none | UAT-04 |

What each role may do in Sprint 1:

| Role | Header links | Create claims | Sees claims |
| --- | --- | --- | --- |
| Requester | My Claims, New Claim | Yes | Own claims only |
| Reviewer | My Claims, New Claim, Review Queue | Yes | Own claims, and claims assigned to them for review |
| Finance Officer | Review Queue | No (no assigned reviewer, BR-15) | Claims assigned to them (the Department Head's) |

A member never reviews their own claim (BR-01): a reviewer's own claim goes to their own assigned reviewer.

### 1.5 Data setup

- No setup by the UAT Team. Each deploy runs the seed, which creates the seven accounts and six sample claims (`CLM-2026-900001` to `CLM-2026-900006`).
- The cases do not use the sample claims, because an earlier tester may have decided them. Each case creates the claims it needs in its first steps.
- Give each claim the title in the case, with the case ID and your run number, for example `UAT-01 run 1 – Khon Kaen client visit`. Use run 2, 3, … when you repeat a case. Then you can always find your own claim in a list.
- All names, amounts, receipt references, and comments in the cases are synthetic. Do not enter real personal or financial data.
- Dates are relative to **D**, the day you run the case in Asia/Bangkok time. `D-3` is three days before D. Enter dates as `YYYY-MM-DD`. Do not run UAT-02 within 10 minutes of midnight Bangkok time, because D changes.
- Money is in Thai baht (THB) with 2 decimals. Enter `1250.50`; the app shows `฿1,250.50`.

### 1.6 Reset and cleanup

- Claims cannot be deleted in Sprint 1, and the claim history is append-only (BR-11). Claims made during UAT stay in the database. This is expected and does not affect a rerun, because each run uses a new title.
- Claims that a case leaves in `Submitted` (UAT-02, 03, 04) stay in a reviewer's queue. Leave them. The queue is oldest first, 20 rows per page: use **Next** to find a newer claim.
- Always log out at the end of a case, and before you log in as another account in the same browser. Or use a separate private window for each account.
- A session ends 8 hours after login. If the app shows the Login screen during a case, log in again with the same account and continue.
- The Product Team does not reset the shared database during the UAT window. **TODO**: agree a reset with the UAT Team only if both teams decide it is necessary.

### 1.7 Case order

Every case is independent: it creates its own data and can run in any order, by either tester. Suggested order: UAT-07, UAT-01, UAT-02, UAT-03, UAT-04, UAT-05, UAT-06. UAT-06 has two sessions; start Session 1 early and run Session 2 at least 1 hour later.

### 1.8 Recording results and reporting findings

- For each case record: case ID, tester name and team, date and time, tested version (the `version` from `/api/v1/health`), the inputs you used, the actual result of each step, and PASS, FAIL, or BLOCKED (with the reason).
- Write what you saw in the actual result. Do not copy the expected result.
- Take a screenshot at each step marked 📷.
- Report each failure with the case ID and step number. **TODO**: agreed channel for findings (GitHub Issue on `beambeambeam/claimflow`, or a shared document).

## 2. Coverage

The categories are the four that the handout requires (§3.1).

| Case | Category | Requirements | Sprint 1 acceptance criteria |
| --- | --- | --- | --- |
| UAT-01 | Normal domain journey | FR-02, FR-03, FR-04, FR-05, FR-06, FR-07, BR-08, BR-17 | F02-AC-01, F02-AC-13, F03-AC-01, F03-AC-05, F03-AC-10, F04-AC-01, F04-AC-03, F04-AC-04, F04-AC-18, F04-AC-20 |
| UAT-02 | Business rule and boundary | BR-05, BR-06, BR-07, BR-12, NFR-08 | F02-AC-03 (UI part), F02-AC-04, F02-AC-05, F02-AC-06, F02-AC-07, F02-AC-11, F02-AC-13 |
| UAT-03 | Forbidden ownership | NFR-02, FR-04, FR-05, BR-02 | F01-AC-18, F03-AC-02, F03-AC-06, F03-AC-07, F03-AC-11, F04-AC-01, F04-AC-08, F04-AC-17 |
| UAT-04 | Forbidden role and self-approval | NFR-02, BR-01, BR-15, FR-05 | F01-AC-20, F02-AC-14, F03-AC-04, F04-AC-02, F04-AC-07 |
| UAT-05 | Business rule: reject reason | BR-04, BR-03, FR-06, FR-07, NFR-08 | F04-AC-05, F04-AC-06, F04-AC-11 (UI part), F04-AC-18 |
| UAT-06 | State, history, and persistence | FR-04, FR-07, NFR-05, BR-03, BR-11 | F03-AC-05, F03-AC-11, F04-AC-04, F04-AC-16 (user view), F04-AC-20 |
| UAT-07 | Login and session | FR-01, NFR-03 | F01-AC-02, F01-AC-03, F01-AC-07, F01-AC-18, F01-AC-19, F01-AC-20 |

Sprint 1 criteria that a user cannot check through the web app are covered by the automated tests in CI (the test tables in each feature contract, and the [traceability matrix](../../traceability.md)). Examples: bans and the 8-hour expiry (F01-AC-04, 05, 08), password and token storage (F01-AC-09, 10), cross-site requests (F01-AC-13), rollback on a failed write (F02-AC-02, F04-AC-14), concurrent decisions and `If-Match` (F04-AC-12, 13), and database-level history protection (F04-AC-15).

## 3. Cases

### UAT-01 Submit a claim and get it approved

| Field | Value |
| --- | --- |
| Category | Normal domain journey |
| References | FR-02 to FR-07, BR-08, BR-17; F02-AC-01, F02-AC-13, F03-AC-01, F03-AC-05, F03-AC-10, F04-AC-01, F04-AC-03, F04-AC-04, F04-AC-18, F04-AC-20 |
| Purpose | A requester submits a claim with two line items; the assigned reviewer approves it; the requester sees the decision and the history |
| Roles | Requester `req.anan`, Reviewer `rev.malee` |
| Preconditions | Logged out. Version recorded |

Inputs:

| Field | Value |
| --- | --- |
| Title | `UAT-01 run <n> – Khon Kaen client visit` |
| Purpose | `Bus and lunch for the client visit` |
| Item 1 | Category Travel, Description `Bus fare, Bangkok to Khon Kaen`, Expense date `D-3`, Amount `1250.50`, Receipt reference `RCPT-UAT01-1` |
| Item 2 | Category Meals, Description `Lunch with the client`, Expense date `D-2`, Amount `980.25`, Receipt reference `RCPT-UAT01-2` |
| Approve comment | `UAT-01 approved` |

Expected total: 1,250.50 + 980.25 = **2,230.75 THB**.

| Step | Action | Expected result |
| --- | --- | --- |
| 1 | Open the URL and log in as `req.anan` | My Claims page. Header shows `Anan K.`, `Requester`, links **My Claims** and **New Claim**, no **Review Queue** |
| 2 | Click **New Claim**. Enter the title, purpose, and Item 1. Click **Add line item** and enter Item 2 | Two line-item rows. The running total shows `Total ฿2,230.75` |
| 3 | Click **Submit** 📷 | Confirmation page: heading `Claim CLM-2026-<6 digits> submitted`, a blue **Submitted** badge, `Total ฿2,230.75`, buttons **View claim**, **View my claims**, **New claim**. Record the claim number |
| 4 | Click **View my claims** | The new claim is the first row: claim number, the title, `฿2,230.75`, **Submitted**, submitted date D (for example `19 Nov 2026`) |
| 5 | Open the claim 📷 | Claim Detail: claim number, **Submitted** badge, title, `฿2,230.75`. Summary: purpose, Requester `Anan K.`, Assigned reviewer `Malee S.`, Submitted D. Line items in order 1, 2 with the inputs. History: one entry, `Submitted`, `– → Submitted`, `Anan K.`, date and time. No **Approve** or **Reject** button |
| 6 | Log out. Log in as `rev.malee`. Click **Review Queue** | The claim is in the queue with Requester `Anan K.`, the title, and `฿2,230.75` (use **Next** if needed). Every row is a `Submitted` claim; no row has Requester `Malee S.` |
| 7 | Open the claim from the queue | Claim Detail with **Approve** and **Reject** buttons |
| 8 | Click **Approve** | Dialog `Approve claim CLM-2026-…?` with an optional comment field, **Approve**, and **Cancel** |
| 9 | Enter the approve comment. Click **Approve** 📷 | Toast `Claim approved.` The badge changes to green **Approved**. **Approve** and **Reject** are gone. History has a second entry: `Approved`, `Submitted → Approved`, `Malee S.`, date and time, `UAT-01 approved` |
| 10 | Click **Review Queue** | The claim is no longer in the queue |
| 11 | Log out. Log in as `req.anan`. Open **My Claims**, then the claim 📷 | The row shows **Approved**. Claim Detail shows **Approved**, total `฿2,230.75`, and two history entries, oldest first: `Submitted` (`Anan K.`) and `Approved` (`Malee S.`, comment `UAT-01 approved`) |

### UAT-02 Claim amount, total, Meals, and date limits

| Field | Value |
| --- | --- |
| Category | Business rule and boundary |
| References | BR-05, BR-06, BR-07, BR-12, NFR-08; F02-AC-03 (UI part), F02-AC-04, F02-AC-05, F02-AC-06, F02-AC-07, F02-AC-11, F02-AC-13 |
| Purpose | Values just past each limit are refused with the field message, values at each limit are accepted, and a refused submission stores nothing |
| Role | Requester `req.ploy` |
| Preconditions | Logged out. Version recorded. Not within 10 minutes of midnight Bangkok time |

Valid inputs (the "baseline"). Each limit is at its highest allowed value:

| Field | Value | Limit |
| --- | --- | --- |
| Title | `UAT-02 run <n> – Regional training trip` | – |
| Purpose | `Travel, hotel, meals, and fee for the regional training` | – |
| Item 1 | Travel, `Return flight`, `D-90`, `20000.00`, `RCPT-UAT02-1` | Oldest allowed date (BR-07); highest item amount (BR-05) |
| Item 2 | Accommodation, `Hotel, 5 nights`, `D-5`, `20000.00`, `RCPT-UAT02-2` | Highest item amount (BR-05) |
| Item 3 | Meals, `Team dinner`, `D`, `1500.00`, `RCPT-UAT02-3` | Today (BR-07); highest Meals amount (BR-12) |
| Item 4 | Training, `Workshop fee`, `D-1`, `8500.00`, `RCPT-UAT02-4` | – |

Expected baseline total: 20,000.00 + 20,000.00 + 1,500.00 + 8,500.00 = **50,000.00 THB**, the highest allowed total (BR-06).

| Step | Action | Expected result |
| --- | --- | --- |
| 1 | Log in as `req.ploy`. Open **My Claims** | No claim with the title `UAT-02 run <n> …` |
| 2 | Click **New Claim**. Enter the baseline, but Item 3 Amount `1500.01` | Running total `Total ฿50,000.01` |
| 3 | Click **Submit** 📷 | No confirmation. Under Item 3 Amount: `A Meals item cannot be more than 1,500.00 THB.` Above **Submit**: `Claim total cannot be more than 50,000.00 THB.` All values stay as entered |
| 4 | Set Item 3 Amount `1500.00`, Item 1 Expense date `D-91`, Item 2 Expense date `D+1`. Click **Submit** 📷 | No confirmation. Under Item 1 Expense date: `Expense date cannot be more than 90 days before the claim was first submitted.` Under Item 2 Expense date: `Expense date cannot be in the future.` No total message |
| 5 | Set Item 1 Expense date `D-90` and Item 2 Expense date `D-5`. Set Item 4 Amount `0.00`. Click **Submit** | No confirmation. Under Item 4 Amount: `Amount must be between 0.01 and 20,000.00 THB.` |
| 6 | Set Item 4 Amount `8500.00` and Item 2 Amount `20000.01`. Click **Submit** | No confirmation. Under Item 2 Amount: `Amount must be between 0.01 and 20,000.00 THB.` Above **Submit**: `Claim total cannot be more than 50,000.00 THB.` (total 50,000.01) |
| 7 | Set Item 2 Amount `1.005`. Click **Submit** | Under Item 2 Amount: `Enter an amount with 2 decimals, e.g. 1250.00.` No confirmation |
| 8 | Set Item 2 Amount `20000.00`. Check that all fields equal the baseline. Click **Submit** 📷 | Running total `Total ฿50,000.00`. Confirmation `Claim CLM-2026-<6 digits> submitted`, **Submitted**, `Total ฿50,000.00`. Record the claim number |
| 9 | Click **New claim**. Click **Add line item** until it is disabled. Count the rows. Then click **Remove** until it is disabled. Do not submit | **Add line item** is disabled at 20 rows. **Remove** is disabled when 1 row is left |
| 10 | Open **My Claims** | Exactly one claim with the title `UAT-02 run <n> …`, total `฿50,000.00`, **Submitted**. The refused attempts in steps 3 to 7 made no claim |
| 11 | Open that claim | Four line items in order 1 to 4 with the baseline categories, dates, and amounts. Total `฿50,000.00`. Assigned reviewer `Malee S.` |

### UAT-03 Other members cannot see or decide a claim

| Field | Value |
| --- | --- |
| Category | Forbidden role and ownership |
| References | NFR-02, FR-04, FR-05, BR-02; F01-AC-18, F03-AC-02, F03-AC-06, F03-AC-07, F03-AC-11, F04-AC-01, F04-AC-08, F04-AC-17 |
| Purpose | Another requester, an unrelated reviewer, and an anonymous visitor cannot see a claim. The claim stays unchanged, and the assigned reviewer can still decide it |
| Roles | Requester `req.anan` (owner), Requester `req.ploy`, Reviewer `rev.somchai` (not assigned), Reviewer `rev.malee` (assigned), anonymous |
| Preconditions | Logged out. Version recorded |

Inputs: Title `UAT-03 run <n> – Office supplies`, Purpose `Paper for the section printer`, one item: Supplies, `Printer paper, 10 reams`, `D-1`, `1200.00`, `RCPT-UAT03-1`.

| Step | Action | Expected result |
| --- | --- | --- |
| 1 | Log in as `req.anan`. Submit a claim with the inputs. Click **View claim** | Claim Detail, **Submitted**, `฿1,200.00`. Record the claim number and copy the page address (`…/claims/<id>`) |
| 2 | Log out. Log in as `req.ploy`. Open **My Claims** | The UAT-03 claim is not in the list |
| 3 | Paste the claim address into the address bar 📷 | Page `You don't have access to this claim.` No claim number, title, amount, line item, or history |
| 4 | Open `<URL>/reviews` | Page `You don't have access to this page.` The header has no **Review Queue** link |
| 5 | Log out. Log in as `rev.somchai`. Paste the claim address 📷 | `You don't have access to this claim.` No claim data |
| 6 | Click **Review Queue** | The UAT-03 claim is not in the queue (it is assigned to `rev.malee`) |
| 7 | Log out. Paste the claim address while logged out | Login screen. The address is `/login?redirect=%2Fclaims%2F<id>` |
| 8 | Log in as `rev.malee` on that screen 📷 | The app opens the UAT-03 claim, not the default page. **Submitted**, `฿1,200.00`, **Approve** and **Reject** buttons. Do not click them |
| 9 | Click **Review Queue** | The UAT-03 claim is in the queue |
| 10 | Log out. Log in as `req.anan`. Open the claim 📷 | Still **Submitted**, `฿1,200.00`. History has one entry (`Submitted`, `Anan K.`). Nothing changed in steps 2 to 9. No **Approve** or **Reject** button for the owner |

Cleanup: the claim stays `Submitted` in `rev.malee`'s queue. Leave it.

### UAT-04 Finance Officer limits and no self-approval

| Field | Value |
| --- | --- |
| Category | Forbidden role and ownership |
| References | NFR-02, BR-01, BR-15, FR-05; F01-AC-20, F02-AC-14, F03-AC-04, F04-AC-02, F04-AC-07 |
| Purpose | A reviewer cannot decide their own claim, which goes to their own reviewer. A Finance Officer cannot create claims, has no My Claims list, and cannot open claims that are not assigned to them |
| Roles | Reviewer `rev.malee` (claim owner), Finance Officer `fin.wanida`, Reviewer `rev.prasert` (Malee's reviewer) |
| Preconditions | Logged out. Version recorded |

Inputs: Title `UAT-04 run <n> – Section meeting snacks`, Purpose `Snacks for the monthly section meeting`, one item: Other, `Snacks for 12 people`, `D-1`, `850.00`, `RCPT-UAT04-1`.

| Step | Action | Expected result |
| --- | --- | --- |
| 1 | Log in as `rev.malee`. Submit a claim with the inputs. Click **View claim** 📷 | **Submitted**, `฿850.00`, Assigned reviewer `Prasert`. No **Approve** or **Reject** button. Copy the page address |
| 2 | Click **Review Queue** | The UAT-04 claim is not in Malee's queue |
| 3 | Log out. Log in as `fin.wanida` | Lands on the Review Queue. Header shows `Wanida`, `Finance Officer`, and only the **Review Queue** link: no **My Claims** and no **New Claim** |
| 4 | Look at every row of the queue | Every row has Requester `Prasert`, or the queue shows `No claims waiting for your review.` No claim of Anan, Ploy, Niran, Malee, or Somchai |
| 5 | Open `<URL>/claims/new` 📷 | Page `You can't create claims.` with the text `Your account has no assigned reviewer, so it cannot submit claims.` No form |
| 6 | Open `<URL>/claims` | Page `You don't have access to this page.` No claim list |
| 7 | Paste the UAT-04 claim address 📷 | `You don't have access to this claim.` No claim data |
| 8 | Log out. Log in as `rev.prasert`. Click **Review Queue** | The UAT-04 claim is in the queue with Requester `Malee S.` |
| 9 | Open it | **Submitted**, `฿850.00`, **Approve** and **Reject** buttons. Do not click them |

Cleanup: the claim stays `Submitted` in `rev.prasert`'s queue. Leave it.

### UAT-05 Reject reason rule and the final state

| Field | Value |
| --- | --- |
| Category | Business rule and boundary |
| References | BR-04, BR-03, FR-06, FR-07, NFR-08; F04-AC-05, F04-AC-06, F04-AC-11 (UI part), F04-AC-18 |
| Purpose | A rejection needs a reason of at least 5 characters. A rejected claim cannot be decided again, and the requester sees the reason |
| Roles | Requester `req.niran`, Reviewer `rev.somchai` |
| Preconditions | Logged out. Version recorded |

Inputs: Title `UAT-05 run <n> – Data course textbook`, Purpose `Textbook for the data handling course`, one item: Training, `Course textbook`, `D-4`, `1450.00`, `RCPT-UAT05-1`. Reject reasons: blank, `Late` (4 characters), `No ID` (5 characters, the space counts).

| Step | Action | Expected result |
| --- | --- | --- |
| 1 | Log in as `req.niran`. Submit a claim with the inputs | Confirmation, **Submitted**, `Total ฿1,450.00`. Record the claim number |
| 2 | Log out. Log in as `rev.somchai`. Open **Review Queue**, then the UAT-05 claim | Requester `Niran`, `฿1,450.00`, **Approve** and **Reject** buttons |
| 3 | Click **Reject** | Dialog `Reject claim CLM-2026-…?` with a Reason field, a counter `0/500`, **Reject**, and **Cancel** |
| 4 | Leave Reason empty. Click **Reject** | Under Reason: `Enter a reason (5 to 500 characters).` The dialog stays open |
| 5 | Enter `Late`. Click **Reject** 📷 | Under Reason: `Reason must be at least 5 characters.` The dialog stays open and `Late` is kept |
| 6 | Change Reason to `No ID`. Click **Reject** 📷 | Toast `Claim rejected.` Red **Rejected** badge. **Approve** and **Reject** are gone. History: `Submitted` (`Niran`), then `Rejected`, `Submitted → Rejected`, `Somchai`, `No ID` |
| 7 | Reload the page | Still **Rejected**, no **Approve** or **Reject** button |
| 8 | Click **Review Queue** | The UAT-05 claim is not in the queue |
| 9 | Log out. Log in as `req.niran`. Open **My Claims**, then the claim 📷 | The row shows **Rejected**. The history shows the `Rejected` entry by `Somchai` with the reason `No ID` |

### UAT-06 Claim state and history stay the same across sessions

| Field | Value |
| --- | --- |
| Category | State, history, and persistence |
| References | FR-04, FR-07, NFR-05, BR-03, BR-11; F03-AC-05, F03-AC-11, F04-AC-04, F04-AC-16 (user view), F04-AC-20 |
| Purpose | A decided claim keeps its status, line items, total, and full history in a later session; the history is in time order and nobody can change it in the app |
| Roles | Requester `req.anan`, Reviewer `rev.malee` |
| Preconditions | Logged out. Version recorded. Session 2 runs at least 1 hour after Session 1, in a new private browser window |

Inputs:

| Field | Value |
| --- | --- |
| Title | `UAT-06 run <n> – Workshop travel` |
| Purpose | `Van, hotel, and lunch for the Q4 workshop` |
| Item 1 | Travel, `Van hire`, `D-10`, `3000.00`, `RCPT-UAT06-1` |
| Item 2 | Accommodation, `Hotel, 2 nights`, `D-9`, `2400.00`, `RCPT-UAT06-2` |
| Item 3 | Meals, `Workshop lunch`, `D-9`, `1499.99`, `RCPT-UAT06-3` |
| Approve comment | `Approved for the Q4 workshop.` |

Expected total: 3,000.00 + 2,400.00 + 1,499.99 = **6,899.99 THB**.

Session 1:

| Step | Action | Expected result |
| --- | --- | --- |
| 1 | Log in as `req.anan`. Submit a claim with the inputs | Running total and confirmation show `฿6,899.99`. Record the claim number |
| 2 | Log out. Log in as `rev.malee`. Open the claim from **Review Queue**. Approve it with the comment | Toast `Claim approved.`, green **Approved** |
| 3 | Log out. Log in as `req.anan`. Open the claim 📷 | **Approved**, `฿6,899.99`, three line items in order 1 to 3, two history entries. Record the date and time of each history entry and the version from `/api/v1/health` |
| 4 | Log out and close the window | – |

Session 2 (at least 1 hour later, new private window):

| Step | Action | Expected result |
| --- | --- | --- |
| 5 | Record the version from `/api/v1/health`. Log in as `req.anan`. Open **My Claims** | The UAT-06 row shows **Approved**, `฿6,899.99`, submitted date D |
| 6 | Open the claim 📷 | Same as step 3: **Approved**; `฿6,899.99`; Item 1 Travel `Van hire` `฿3,000.00`, Item 2 Accommodation `Hotel, 2 nights` `฿2,400.00`, Item 3 Meals `Workshop lunch` `฿1,499.99`, with the input dates and receipt references; Assigned reviewer `Malee S.`. History, oldest first: `Submitted` (`– → Submitted`, `Anan K.`) and `Approved` (`Submitted → Approved`, `Malee S.`, `Approved for the Q4 workshop.`), with the same dates and times as step 3 |
| 7 | Look for any control that changes the claim or its history | None: no edit, delete, withdraw, approve, or reject control |
| 8 | Log out. Log in as `rev.malee`. Open the claim by its address, then open **Review Queue** | Claim shows **Approved** with no **Approve** or **Reject** button. The claim is not in the queue |

If the version in step 5 is different from step 3, the claim survived a redeploy too. Record both versions.

### UAT-07 Log in, role landing pages, and log out

| Field | Value |
| --- | --- |
| Category | Supporting case: access control at login |
| References | FR-01, NFR-03; F01-AC-02, F01-AC-03, F01-AC-07, F01-AC-18, F01-AC-19, F01-AC-20 |
| Purpose | Only valid credentials open a session, a failed login does not show which part was wrong, each role lands on its own page, and logout ends the session |
| Roles | Anonymous, Requester `req.anan`, Reviewer `rev.malee` |
| Preconditions | Logged out. Version recorded |

Inputs: wrong password `uat-wrong-password`; unknown username `nobody.uat`; username in mixed case `REQ.Anan`.

| Step | Action | Expected result |
| --- | --- | --- |
| 1 | Open `<URL>/claims` while logged out | Login screen. Address `/login?redirect=%2Fclaims` |
| 2 | Click **Log in** with both fields empty | Under Username `Enter your username.` Under Password `Enter your password.` |
| 3 | Enter username `req.anan` and the wrong password. Click **Log in** 📷 | Form message `Invalid username or password.` Username kept, password field empty |
| 4 | Enter username `nobody.uat` and the wrong password. Click **Log in** | The same message `Invalid username or password.` |
| 5 | Enter username `REQ.Anan` and the correct password. Click **Log in** | My Claims page (`/claims`). Header `Anan K.`, `Requester`, links **My Claims** and **New Claim** |
| 6 | Click **Log out** | Login screen |
| 7 | Press the browser **Back** button, or open `<URL>/claims` | Login screen again; no claim data shows |
| 8 | Open `<URL>/login`. Log in as `rev.malee` | Lands on the Review Queue (`/reviews`). Header `Malee S.`, `Reviewer`, links **My Claims**, **New Claim**, **Review Queue** |
| 9 | Click **Log out** | Login screen |

## 4. Open items

| Item | Owner |
| --- | --- |
| Date the password was posted in `#qa` (§1.4) | Product Team |
| UAT session dates: first run, defect fixes, retest (§1.1) | Both teams |
| Channel for findings (§1.8) | Both teams |
| Reset of the shared database, only if both teams decide it is necessary (§1.6) | Product Team |
