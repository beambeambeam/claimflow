> **LOCAL run, not the F10-TEST-12 evidence.** F10-AC-12 needs a run against the deployed release with its tag. This one is a dry run of the script on a developer machine: a throwaway `postgres:17` container, the API started with `go run .` (not a release build, `APP_VERSION=dev`), the Bun script on the same host, so no network between them, and `BCRYPT_COST=10`. The numbers show that the script and the load data work. They do not show NFR-07 on the deployed instance.

- **Commit:** `fcebe10` on `feat/load-dataset` (run as `39f8ca9`, the same code before the branch was rebased onto `sprint-3`)
- **Data:** the base seed and `seed-load` (240 load claims), freshly loaded; 86 `SUBMITTED` load claims assigned to `rev.malee` before the run, 20 approved by it
- **Command:** `bun run perf -- --base-url http://127.0.0.1:8080 --rounds 20`, with `SEED_USER_PASSWORD` set in the environment

# NFR-07 timed run

- Base URL: http://127.0.0.1:8080
- Time: 2026-11-27T12:40:21.070Z
- API version: dev
- Rounds: 20 timed, after 1 warm-up round that is not counted
- Result: PASS, overall p95 83.9 ms (limit 1000 ms)

| Request | Count | p50 (ms) | p95 (ms) | max (ms) |
| --- | --: | --: | --: | --: |
| POST /auth/login (req.anan) | 20 | 77.8 | 84.5 | 89.4 |
| POST /auth/login (rev.malee) | 20 | 78.0 | 88.5 | 90.4 |
| POST /auth/login (fin.wanida) | 20 | 79.0 | 85.7 | 87.8 |
| GET /auth/me | 20 | 2.5 | 2.7 | 2.9 |
| GET /claims?scope=mine | 20 | 6.4 | 7.1 | 7.3 |
| GET /claims?scope=review | 20 | 6.3 | 7.1 | 7.4 |
| GET /claims?scope=finance | 20 | 5.9 | 6.5 | 6.7 |
| GET /claims?scope=all | 20 | 6.9 | 7.4 | 8.2 |
| GET /claims?scope=all (status, category, range, q) | 20 | 7.4 | 8.0 | 9.2 |
| GET /claims/{id} | 20 | 7.2 | 8.8 | 9.2 |
| POST /claims/{id}/approve | 20 | 16.6 | 23.3 | 25.5 |
| GET /reports/summary | 20 | 8.1 | 10.1 | 10.4 |
| GET /reports/summary (range) | 20 | 8.3 | 10.6 | 10.9 |
| **Overall** | 260 | 7.7 | 83.9 | 90.4 |
