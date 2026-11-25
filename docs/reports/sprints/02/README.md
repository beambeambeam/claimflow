# Sprint 2 report evidence index

Project Deliverable 2 (CPE 334), due 2026-11-29 23:59 on LEB2. Build with `bun run pdf 02` in `docs/reports` (see `docs/reports/AGENTS.md`). Candidate values are marked `\todo{recapture on candidate: ...}` in the sources and must be recaptured on the submitted commit before the DRAFT box is removed.

| Part | Artifact | IDs / version | Result |
| --- | --- | --- | --- |
| Front | `content/00-front.tex` | tag `v0.1.1.0`, commit `3a91ea4` | navigation table, deployed URL |
| A.1 | `content/part-a-1-continuity.tex`, `figures/demo/a1-*`, `demo.ts` | `v0.0.3.0` (`ac5bce5`) to `v0.1.1.0` | Sprint 1 journey and BR-12 pass |
| A.2, A.3 | `content/part-a-2-register.tex` | #86, #87, #99, #66, #67, #70, #61 | 0 open; #61 needs a rationale |
| B.1 | `content/part-b-1-contracts.tex` (generated) | F-05 0.3, F-06 0.4, F-07 0.2, DoD | 29 figures |
| B.2 | `content/data/completion-table.tex` (generated) | 43 AC | 43 pass |
| B.3 | `content/part-b-3-demos.tex`, `figures/demo/d*`, `content/data/db-readback.txt` | F05-TEST-05, F06-TEST-07, F07-TEST-04 | as expected |
| B.4 | `content/part-b-4-tests.tex`, `content/data/ac-table.tex`, `tdd-*.txt`, `excerpt-*` | 56 tests; BR-14 `a547ac0` to `71a8e43`; BR-16 `53e8827` to `38e0165` | Go 178 + 241 subtests pass; web 182 pass in CI; E2E 8 pass |
| B.5 | `content/part-b-5-workflow.tex`, `figures/github/*` | #108, #104, #109 | merged to `sprint-2`, then `main` |
| C | `content/part-c-sprint3.tex` | F-08, F-09, F-10 | skeleton; `\todo{beam: ...}` |
| D | `content/part-d-*.tex` | SRS 0.3, SDS, ADR-0001, traceability | 41 requirements, 0 uncovered |
| E | `content/part-e-*.tex` | CI 45614889848, Deploy demo 45615736377 | all jobs green; ready for Sprint 2 only |
| Back | `content/back-matter.tex`, `content/data/team.tex` | contributions since 2026-11-16 | AI-use of student two is a todo |
