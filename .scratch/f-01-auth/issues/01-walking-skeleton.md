# 01: Walking skeleton: contract, database, codegen, health and readiness

**What to build:** The API and web run on one shared foundation. The API has a hand-written OpenAPI contract with a generated strict server, a PostgreSQL database with versioned migrations, generated type-safe queries, RFC 9457 problem responses with a request ID, and a test harness that starts a real PostgreSQL. The web has a generated API client with TanStack Query options and Zod schemas. `GET /health` answers without the database, and `GET /ready` answers 200 when the database is reachable and 503 when it is not.

**Blocked by:** None (can start immediately). Note: the #6 contract for F-01 merges before the implementation PR.

**Status:** ready-for-agent

- [ ] Local PostgreSQL 17 starts with one command, and migrations run with one command.
- [ ] The contract defines `/health`, `/ready`, and the shared `Problem` schema; the API implements the generated interface.
- [ ] `/health` returns `{ status, version }` with no database access; `/ready` returns 503 when the database is down.
- [ ] Every error body uses the problem format with `code` and `requestId`; no stack traces, SQL, or Go error strings.
- [ ] The web uses the generated client and TanStack Query through the router context.
- [ ] CI regenerates API and web code and fails on any diff.
- [ ] New API variables (`DATABASE_URL`, `COOKIE_SECURE`, `BCRYPT_COST`, `LOG_LEVEL`, `APP_VERSION`) are in the varlock schema, and a committed API example environment file lists them with safe values.
- [ ] Integration tests run against a real PostgreSQL started by the harness; generated code is excluded from coverage; 100% coverage holds.
