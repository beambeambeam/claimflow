# 02: Load dataset and NFR-07 timed run

**What to build:** An operator loads at least 200 extra synthetic claims with one command and runs a repeatable timed test of the main API requests against the deployed instance. The result proves that 95% of requests finish in under 1 second (NFR-07) and is recorded with the tested release.

**Blocked by:** F-09/01 All-claims view with filters and search; F-10/01 Decision summary screen. External: GCP deployment from Lab 5.

**Status:** ready-for-agent

- [ ] `seed:load` adds ≥ 200 claims across statuses, categories, and months, with history, idempotent by claim number.
- [ ] Indexes are checked against the list, filter, and summary queries.
- [ ] A scripted run covers login, `/auth/me`, each list scope, filters, detail, an action, and the summary; it reports p95 per request and overall.
- [ ] The run on the deployed instance shows p95 < 1 s; results are recorded with the release tag and commit.
