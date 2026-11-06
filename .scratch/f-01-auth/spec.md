# F-01: Authentication and session

Status: ready-for-agent Sprint: 1 Requirements: FR-01, NFR-01, NFR-03 Establishes: SR-01 (authorization enforcement), SR-02 (secrets, local), SR-03 (persistence, local), SR-06 (tests and CI gate) Depends on: none Design: SDS §2.2 `user`, `account`, `session`; §2.5; §4.1–4.2; §5.1; §6; §7.1 Login; §8

## Problem Statement

ClaimFlow has no way to know who is using it. Without identity, the system cannot tell a requester from their assigned reviewer, so it cannot block self-approval, keep claims private, or record who made a decision. The repo also has no database, no API contract, and no generated client yet, so no feature can persist anything or be tested against real storage.

## Solution

Members log in with a seeded username and password and get a server-side session in an HttpOnly cookie. Every protected operation reads the actor only from that session. The system returns the current user's identity, role, and permissions, so the web shows only the links the role allows. Logout ends the session at once, and a session also ends 8 hours after login. A banned member cannot log in and loses all sessions.

This feature also lays the foundation that every later feature uses: PostgreSQL with versioned migrations, reproducible seed data, the hand-written OpenAPI contract with generated server and web client, RFC 9457 problem responses, and a real-database test harness.

## User Stories

1. As an anonymous visitor, I want to see a login page, so that I can sign in to ClaimFlow.
2. As an anonymous visitor, I want every protected page to send me to the login page, so that I never see private data without signing in.
3. As an anonymous visitor, I want to return to the page I asked for after I log in, so that I do not have to navigate again.
4. As a member, I want to log in with my username and password, so that the system knows who I am.
5. As a member, I want my username to work in any letter case, so that a capital letter does not lock me out.
6. As a member, I want a wrong password and an unknown username to give the same message, so that nobody can find out which accounts exist.
7. As a member, I want my session kept in a cookie that scripts cannot read, so that a script on the page cannot steal it.
8. As a requester, I want to land on My Claims after login, so that I start where my work is.
9. As a reviewer or Finance Officer, I want to land on the Review Queue after login, so that I start with the claims waiting for me.
10. As a member, I want the header to show my display name and role, so that I know which account I am using.
11. As a member, I want the header to show only the links my role allows, so that I am not offered screens I cannot use.
12. As a Finance Officer, I want no "My Claims" or "New Claim" links, so that I am not offered actions I can never take (BR-15).
13. As a member, I want to log out, so that nobody else can use my session on a shared computer.
14. As a member, I want my old session token to stop working right after logout, so that a copied cookie is useless (NFR-03).
15. As a member, I want my session to end 8 hours after login, so that an abandoned session does not stay open forever (NFR-03).
16. As a member whose session ended, I want to be sent to the login page with a clear state, so that I can sign in again without seeing stale data.
17. As a department administrator, I want a banned member to be unable to log in, so that I can deactivate an account through seed data or a migration.
18. As a department administrator, I want a ban to end the member's existing sessions, so that deactivation takes effect at once.
19. As a department administrator, I want a ban with an expiry date in the past to count as no ban, so that a temporary ban ends by itself.
20. As a security reviewer, I want passwords stored only as bcrypt hashes with cost of at least 10, so that a database leak does not expose passwords (NFR-01).
21. As a security reviewer, I want no password or session token in logs or API responses, so that logs are safe to share (NFR-01).
22. As a security reviewer, I want only a hash of the session token stored, so that a database leak gives no usable sessions.
23. As a security reviewer, I want a state-changing request from a foreign origin rejected, so that another site cannot act as a logged-in member.
24. As a security reviewer, I want every protected operation to return 401 without a session, so that no endpoint is left open by mistake (NFR-02).
25. As a security reviewer, I want a route permission check that returns 403 when the role does not grant the operation, so that role rules live in one place (NFR-02).
26. As a developer, I want a test that walks every operation in the contract and checks 401 without a session, so that a new operation is covered without a new test.
27. As a developer, I want the identity of the actor to come only from the session, so that no request can pretend to be someone else by sending a user ID or role.
28. As a developer, I want one OpenAPI contract that generates the API server interface and the web client, so that the two apps cannot drift apart.
29. As a developer, I want CI to fail when generated code is out of date, so that the contract and the code always match.
30. As a developer, I want every error returned as an RFC 9457 problem with a stable `code`, so that the web can handle errors in one way.
31. As a developer, I want versioned SQL migrations, so that every environment has the same schema.
32. As a developer, I want an idempotent seed command that loads the reporting line and demo accounts, so that every developer and test uses the same people.
33. As a developer, I want the seed password to come from an environment variable, so that no real password is committed (NFR-04).
34. As a developer, I want the API to refuse to start when a required environment variable is missing or invalid, so that misconfiguration fails fast.
35. As a developer, I want a committed example environment file for the API, so that a new developer knows every variable to set (NFR-04).
36. As a developer, I want integration tests to run against a real PostgreSQL started by the test harness, so that tests cover real constraints and transactions.
37. As an operator, I want a liveness endpoint that does not touch the database, so that a platform can tell the process is up.
38. As an operator, I want a readiness endpoint that returns 503 when the database cannot be reached, so that traffic goes only to working instances.
39. As an operator, I want data to remain after the API restarts, so that sessions and later claims are durable (NFR-05).
40. As a member, I want a loading state while the login request runs and a disabled button, so that I do not submit twice.

## Implementation Decisions

- **Foundation, built first.** PostgreSQL 17 for local development, `goose` SQL migrations, `sqlc` over `pgx/v5` generating one shared query package, `oapi-codegen` generating the Gin strict server interface and types, and `@hey-api/openapi-ts` generating the web fetch client, TanStack Query options, and Zod schemas. Generated code is committed, and CI regenerates it and fails on any diff. Tools are pinned through `go tool`, never in the main module file.
- **One API contract.** A hand-written OpenAPI 3.0.3 file is the source of truth. This feature adds `POST /auth/login`, `POST /auth/logout`, `GET /auth/me`, `GET /health`, and `GET /ready`, plus the shared `Problem` schema. The server composes one struct that implements the generated interface; each feature's handler embeds into it.
- **Problem responses.** One shared package produces every error body: `type`, `title`, `status`, `code`, `instance`, `requestId`, and optional `errors[]` with `field`, `rule`, `message`. No stack traces, SQL, or Go error strings leave the server. Each request gets a request ID, logged with `slog` JSON.
- **Schema.** Three auth tables keep concerns apart: `user` (identity, single role from `requester`/`reviewer`/`finance`, `reviewer_id` for the reporting line with CHECK `reviewer_id <> id`, ban fields, lowercase unique username), `account` (one `credential` row per user with the bcrypt hash), and `session` (SHA-256 of the token, `expires_at`, IP and user agent). UUIDv7 primary keys generated in Go.
- **Auth module.** Owns login, logout, session lookup, and ban checks. Login lowercases the username, compares against the stored hash, and compares against a dummy hash for unknown users to limit timing differences. A banned user gets the same generic 401. Expired sessions are deleted at login. Setting a ban deletes the user's sessions.
- **Session.** 32 random bytes, base64url, as the cookie value; only its SHA-256 is stored. Fixed 8-hour lifetime, no sliding renewal; the lifetime is a constant, not configuration. Cookie `__Host-sid` with `HttpOnly`, `Secure`, `SameSite=Lax`, `Path=/`; local HTTP uses `sid` without `Secure` when `COOKIE_SECURE=false`.
- **Middleware order.** Cross-origin protection (standard library `CrossOriginProtection`) → session → actor in request context (401 on missing, unknown, expired, or banned) → OpenAPI request validation (400) → route permission (403).
- **Route permissions.** A statement of resources and actions (`claim: create, list`; `review: list`; `finance: list`; `report: read`) and a per-role subset, checked by a `RequirePermission(resource, action)` middleware. Code checks permissions, never role names. `GET /auth/me` returns `{ id, username, name, role, permissions }`.
- **Clock.** The auth module takes the current time from an injected clock, so expiry is testable without waiting.
- **Configuration.** New API variables in the varlock schema: `DATABASE_URL` (sensitive), `COOKIE_SECURE`, `BCRYPT_COST` (default 12, minimum 10), `LOG_LEVEL`, `SEED_USER_PASSWORD` (sensitive, seed only), `APP_VERSION`. A committed API example environment file lists all of them with safe values.
- **Seed.** Idempotent, upserts by username. Seven users form the reporting line: `req.anan`, `req.ploy`, `req.niran`, `rev.malee`, `rev.somchai`, `rev.prasert`, `fin.wanida` (SDS §2.5). All share `SEED_USER_PASSWORD`.
- **Web.** TanStack Query client in router context. A Login route, and an `_auth` layout route whose `beforeLoad` ensures `/auth/me` and redirects anonymous users to `/login?redirect=<path>`. A header with display name, role, links from `permissions`, and Log out. A 401 anywhere clears the query cache and redirects to login. Post-login redirect: requesters to `/claims`, reviewers and Finance to `/reviews` (those screens arrive in F-03 and F-04; until then they can be placeholders inside `_auth`).

## Testing Decisions

- Good tests check behaviour through a module's public interface, not its internals, so a refactor does not break them.
- **Auth module against real PostgreSQL** (harness that starts a container, runs migrations, and seeds): valid login, wrong password, unknown user, case-insensitive username, banned user, expired ban, ban deletes sessions, logout invalidates the token, session expiry with a fixed clock, password stored as a bcrypt hash with cost ≥ 10.
- **HTTP through the full router** with `httptest`: cookie attributes, generic 401 message, `/auth/me` shape and permissions per role, cross-origin state change → 403, problem response shape, `/health` without DB, `/ready` 503 when the DB is down.
- **Route inventory test**: reads every operation from the contract and asserts 401 without a session, except `/auth/login`, `/health`, and `/ready`. Later features are covered automatically.
- **Web component tests** (Vitest, Testing Library, MSW handlers typed from the generated client): login form errors and loading state, redirect after login by role, header links from permissions, Finance sees no claim links, 401 redirect.
- **Coverage**: 100% statement coverage in both apps, with generated code excluded (the generated API and DB packages are added to the API coverage exclusions).
- Prior art: the existing config and server tests in the API, and the router and home tests in the web.

## Out of Scope

- Self-registration, password recovery, email verification (EX-04).
- Account administration UI: creating users, changing roles, banning through the UI (EX-08).
- Docker compose for the whole app, GCP deployment, Secret Manager (Lab 5).
- Claim screens and claim endpoints (F-02 onward).

## Further Notes

- The engineering contract for this feature (#6) must merge before the implementation PRs, so the history shows spec → test → code.
- Sprint 1 integration branch is `staging-1` (SDS §9). Issues #5 and #7 say `sprint-N`; the SDS and the remote use `staging-N`.
- Use TDD for rule work and keep the red commit and green commit for the D1 report (#4).
