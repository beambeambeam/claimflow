# Product Definition of Done – Sprint 1

- **Applies to:** every Sprint 1 feature (F-01, F-02, F-03, F-04) and every implementation PR for them
- **Issue:** #6 · **Version:** 0.1 (draft) · **Last updated:** 2026-11-05

A feature is done only when every item below is true **and** has the evidence named in the right column. A statement from a person or an AI agent that something "works" or "passes" is not evidence. Evidence is a link, a commit ID, a CI run, a file path, or a recorded result that a reviewer can open and check.

Each feature contract adds its own items in section 7 (Feature DoD). Both lists apply.

## 1. Checklist

| # | Item | Evidence |
| --- | --- | --- |
| 1 | **Behavior implemented.** Every AC in the feature contract is met. Nothing in the contract is left out without a note in item 10. | Merged PR that links the contract and lists its AC IDs. |
| 2 | **Contract first.** The contract (and any revision) was merged before the implementation PR. | `git log --oneline -- docs/sprints/sprint-1/<feature>.md` shows the contract commit before the implementation commits. |
| 3 | **Planned tests executed.** Every TEST row in the STS is implemented and run at its stated level. Manual tests are run and recorded. | Test file and test name per TEST ID in `docs/traceability.md` (Actual test column); manual results recorded with the commit ID they ran on. |
| 4 | **Meaningful tests passing.** All automated tests pass in CI on the merge commit. Go coverage is 100% (`bun run test:coverage` in `apps/api`). Tests check the outcomes in the AC, not only that code runs. | Green CI run linked from the PR; Result and Version columns in `docs/traceability.md` set to `pass` and the commit or tag. |
| 5 | **TDD for rule work.** For business rules (BR-xx), the failing test was committed before the code that makes it pass. | Commit IDs of the red test and the green implementation, noted for the Sprint 1 report (#4). |
| 6 | **Review and integration complete.** The PR is reviewed and approved by the other student, merged into the integration branch with a squash merge, and `bun run check` passes at the root. | PR approval, merge commit on `sprint-1`, the required CI checks (`code_quality`, `test`, `api`, `build`) green. |
| 7 | **Access rules verified.** Every new protected endpoint returns 401 without a session and 403 for a role or user without access (NFR-02). | Integration test names for the 401 and 403 cases. |
| 8 | **Setup and documents current.** `README`, `.env.example`, OpenAPI file, migrations, seed, and `docs/traceability.md` match the merged code. A new developer can run the app from the README. | Diff in the PR; the README steps run on a clean checkout. |
| 9 | **No secrets committed.** No passwords, keys, or tokens in Git (NFR-04). | F01-TEST-18 result recorded for the tested commit (tracked `.env` files and secret scan). |
| 10 | **Incomplete work disclosed.** Any AC not met, test skipped, known bug, or limitation is written down. Nothing is hidden or marked done when it is not. | "Known limitations" in the PR body and in the Sprint 1 report (#4); a follow-up issue for each item. |

## 2. When a contract changes

If an AC or test changes after implementation starts, add a row to the contract's revision history with the reason. Do not rewrite earlier versions. Update `docs/traceability.md` in the same PR.

## 3. Revision history

| Version | Date       | Change              | Author     |
| ------- | ---------- | ------------------- | ---------- |
| 0.1     | 2026-11-05 | Initial Product DoD | 1005, 1045 |
