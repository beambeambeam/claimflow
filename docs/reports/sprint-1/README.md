# Sprint 1 report (Deliverable 1)

LaTeX source of the Deliverable 1 PDF: one self-contained document with Parts A–F, built with XeLaTeX. Issue #4 owns it. The specification is `docs/project-specifications/project-deliverable-1.md`; where this file and the specification differ, the specification wins.

## What goes in the PDF

| Part | Points | Content | Source |
| --- | --- | --- | --- |
| Front matter | – | Both students' names and IDs, Theme 5 ClaimFlow, repository URL, release tag and commit, contents with page numbers, navigation table (one row per part: page range and main IDs) | `content/00-front.tex` |
| A | 10 | Complete SRS | capture of `docs/srs.md` |
| B | 10 | Feature inventory and traceability | captures of `docs/features.md`, `docs/traceability.md` |
| C | 15 | System-level SDS and the ADRs | captures of `docs/system-sds.md`, `docs/adr/*.md` |
| D | 5 | Three-sprint plan | capture of `docs/sprints/plan.md` |
| E | 20 | Every Sprint 1 contract and the Product DoD | captures of `docs/sprints/sprint-1/*.md` |
| F | 20 | Execution evidence, written for the report | `content/part-f-*.tex` |

Parts A–E are the repo documents as GitHub renders them, captured by script and cut to pages, so the PDF and the repo cannot drift. Part F is written in LaTeX from evidence that scripts collect.

## Where things are

`main.tex` inputs `content/00-front.tex`, the generated figure lists `content/part-a…e-*.tex`, and the hand-written `content/part-f-*.tex`. `content/data/` holds generated text evidence; `figures/` holds every image. `report.config.ts` lists what the scripts capture for this sprint; `demo.ts` drives the three demos. Shared scripts, the preamble, setup, and build steps: [`../README.md`](../README.md). The submitted PDF goes to LEB2.

## Rules of the report

- **One commit for every capture.** Screenshots and test output come from the release commit on `main`. Before the release exists, a draft may capture from `sprint-1`; the front matter then says so in red, and the final build recaptures.
- **A missing figure never breaks the build.** `\evidence` draws a red "pending" box when its file does not exist. `check-report` counts those boxes and every `\todo`; the final PDF has none.
- **Scripts capture, people do not.** A rerun refreshes the report. The only hand capture is the project board.
- **Terminal output is text** in a `shell` block, never a screenshot.
- **Every figure is readable at normal zoom.** A capture taller than one page is cut on a blank row into pieces.
- **Automated tests and manual checks are separated** in the AC-to-test table. Failed, skipped, and unimplemented rows are shown as they are.
- **No secrets.** Seed account names appear; passwords do not.

## Evidence index

**15 November 2026: phase 3 is done.** Every row below is filled from the release `v0.0.3.0` (`ac5bce5`) except the AI-use declaration and the contribution check of 1045 in F.5. The statuses below are the plan as written in phase 1.

Status: `ready` can be written now from merged work, `release` waits for the Sprint 1 release on `main`, `blocked` waits for a ticket.

| Part | Artifact | IDs / version | Status |
| --- | --- | --- | --- |
| Front | Names, IDs, theme, repo URL | – | done |
| Front | Release tag and commit | `vG.X.Y.Z` | release |
| A–E | Document captures | `main` at the release commit | release (draft from `sprint-1` now) |
| F.1 | Setup summary | README at the release commit | ready (recheck at release). Finding: at `a60fcf1` the README does not explain the seed or `SEED_USER_PASSWORD`, which the Product DoD asks for |
| F.2 | Demo 1: submit, approve, retrieve the saved result | F-02, F-03, F-04 | blocked by #30, #31, #32 |
| F.2 | Demo 2: rule rejection under the field (Meals limit, future date) | F02-AC-06, F02-AC-07, BR-07, BR-12 | release (script ready now) |
| F.2 | Demo 3: login, logout, forbidden operation with API test evidence | F-01, F02-AC-09, F02-AC-14, BR-15 | release (script ready now) |
| F.3 | Build and test commands with pass, fail, skip counts | release commit | release |
| F.3 | AC to planned test to actual test to result | `docs/traceability.md` | ready for F-01, F-02; blocked for F-03, F-04 |
| F.3 | Excerpt: domain rule | BR-07 or BR-12, `apps/api/internal/claim/rules_test.go` | ready |
| F.3 | Excerpt: role or ownership rule | F02-TEST-14 (`apps/api/claims_create_test.go`), or an F-04 ownership test once it exists | ready |
| F.3 | TDD red and green | red `ca3a703`, green `7b4787d` (PR #60, F02-TEST-04, 06, 08, 09, 11) | ready |
| F.4 | Chain 1: #27 → `feat/route-permissions` → PR #59 | F-01/04; author 1005, reviewer 1045; review found the session check failing open | ready |
| F.4 | Chain 2: #25 → `feat/login-current-user` → PR #56 | F-01/02; author 1045, reviewer 1005; changes requested with six findings | ready |
| F.4 | Both chains reaching `main` | release PR | release |
| F.4 | History: contracts before implementation | PR #23 (2026-11-06) before PRs #36, #56, #58, #59, #60 | ready |
| F.4 | Final board screenshot and release reference | project 5 | release |
| F.5 | Contribution table | merged PRs and closed issues per student | ready (update at release) |
| F.5 | AI-use declaration | tools and tasks, both students | ready for 1005; 1045 to supply |
| F.5 | Known limitations, failed and skipped tests | contract known gaps, review follow-ups | ready (update at release) |

Squash merges keep one commit per PR on `sprint-1`. The red and green commits of a PR stay reachable on GitHub under the PR's Commits tab and as `refs/pull/<n>/head`; the report cites them with the PR number.

## Phases

1. **Skeleton and static evidence** (now). Preamble, `main.tex`, every content file, the scripts, a draft capture of the documents from `sprint-1`, and every `ready` row above. Done when: XeLaTeX builds twice with no error, the contents and the navigation table show page numbers, and `check-report` lists each remaining pending figure and `\todo` with the row it waits for.
2. **Sprint 1 completes.** #30–#34 merge; the release PR `sprint-1` → `main` merges; the tag exists.
3. **Release evidence.** Recapture documents from `main`, run the demo script and the test commands on the release commit, capture the board, fill the tag and commit. Done when: `check-report` reports zero pending figures and zero `\todo`, and every acceptance criterion of #4 is ticked against a page of the PDF.
4. **Submit** the PDF on LEB2. Nothing is committed afterwards that changes the tagged release.

## Build

See [`../README.md`](../README.md); this sprint is `sprint-1`. `content/data/` (`ac-table.tex`, `team.tex`, `history.txt`, the two TDD outputs, the two test excerpts) is committed because the commands behind it need Docker and Go; regenerate it on the release commit.
