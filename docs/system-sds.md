# System-Level Software Design Specification (SDS)

- **Product:** ClaimFlow – Department Expense Reimbursement System (repository: `beambeambeam/claimflow`)
- **Team:** 1005, 1045
- **Version:** 0.21 (draft for Deliverable 1)
- **Last updated:** 2026-11-28
- **Baseline:** `docs/srs.md` v0.4 · Glossary: `GLOSSARY.md` · Decisions: `docs/adr/`

This document defines the system-wide design: architecture, data model, the API contract and its conventions, roles and access, authentication, configuration, and deployment. Each feature contract in `docs/sprints/` refines it with exact operations in `apps/api/openapi.yaml`, screens, and schema changes.

---

## 1. Architecture

### 1.1 Overview

ClaimFlow is a **modular monolith**. It lives in a Bun workspace monorepo: `apps/web` is the UI, `apps/api` is the API and application logic, and one OpenAPI contract (`apps/api/openapi.yaml`) defines the interface between them.

```mermaid
flowchart LR
    B[Browser<br/>React SPA<br/>apps/web] -- "HTTPS, JSON<br/>session cookie<br/>same origin /api" --> API

    subgraph C[API container – apps/api]
        API[Gin router<br/>/api/v1/*<br/>+ static web files] --> MW[Cross-origin protection<br/>session middleware<br/>OpenAPI request validation]
        MW --> H[Handlers<br/>generated strict server interface]
        H --> CM[Claim module<br/>policy + lifecycle + rules<br/>+ Receipt Files]
        H --> AM[Auth module]
        CM & AM --> DB[internal/db<br/>sqlc over pgx]
        CM --> ST[internal/storage<br/>aws-sdk-go-v2 S3 client]
    end

    DB -- "SQL over TCP" --> PG[(PostgreSQL)]
    ST -- "HeadObject, GetObject range,<br/>DeleteObject, presign" --> S3[(S3 bucket<br/>RustFS locally, AWS S3 in production)]
    B -- "PUT / GET with presigned URL<br/>(Receipt File bytes only)" --> S3
    SPEC[(openapi.yaml)] -. generates .-> H
    SPEC -. generates .-> CL[web API client] -.-> B
```

- **Local:** the web dev server (`:3000`) proxies `/api` to the API (`:8080`), so the browser stays on one origin. This proxy already exists in the repo.
- **Production:** the API container also serves the built web files (Section 9), so there is one origin and cookies work without any CORS setup.
- **Receipt Files (F-11):** the browser sends and fetches Receipt File bytes straight to and from the S3 bucket with presigned URLs that the API signs after its own checks. The API never carries file bytes (ADR-0002). Only the bucket needs CORS; the app still needs none.

### 1.2 Technology stack

| Layer | Choice | Status | Reason |
| --- | --- | --- | --- |
| Monorepo/tooling | Bun workspaces, Vite+ (`vp`) for lint/format/test, semantic-release | in repo | Existing team setup |
| UI | React 19 + TypeScript, TanStack Router (file routes) | in repo | Existing team setup; typed routes |
| UI components | Astryx (`@astryxdesign/core`, precompiled CSS) with the Neutral theme | in repo | Accessible ready-made components and layout, themed through tokens, with no CSS build step |
| UI server state | TanStack Query | to add | Caching, plus built-in loading/error states for every API call |
| UI forms | TanStack Form with the Zod schemas generated from the contract; `#/lib/form` binds Astryx inputs to fields | in repo | Client checks use the same limits as the server |
| API contract | OpenAPI 3.0.3, written by hand in `apps/api/openapi.yaml` | to add | One source for both apps; it is the feature contract for each sprint |
| API code generation | `oapi-codegen` (Gin strict server + types), pinned in `tools.mod` | to add | Handlers implement a generated interface, so the code cannot drift from the contract |
| Web code generation | `@hey-api/openapi-ts` (fetch client, TanStack Query options, Zod schemas) | to add | Typed calls and query keys with no hand-written fetch code |
| API | Go + Gin | in repo | Existing team setup; single static binary |
| Validation | Request shape: OpenAPI request validator middleware (`oapi-codegen/gin-middleware`). Business rules: the claim module. | to add | Shape errors and business-rule errors stay separate |
| Database | PostgreSQL 17 | to add | Relational constraints and transactions; Cloud SQL on GCP |
| DB driver / queries | `pgx/v5` + `sqlc` (SQL → type-safe Go) | to add | Plain SQL with compile-time types, no ORM |
| Migrations | `goose` (SQL files), pinned in `tools.mod` | to add | Versioned migrations, runnable as a `go tool` |
| IDs | UUIDv7, generated in Go (`google/uuid` `NewV7`) | to add | Time-ordered keys keep index inserts local; works on any PostgreSQL version |
| Auth schema | `user`, `account`, and `session` tables, implemented in Go (Section 2.2) | to add | Keeps identity, credentials, and sessions apart, so a new sign-in method adds rows, not columns |
| Password hashing | `golang.org/x/crypto/bcrypt`, cost 12 | to add | Meets NFR-01 |
| CSRF | `net/http.CrossOriginProtection` (Go 1.25+) | to add | Standard library check that uses the browser's Fetch metadata; no tokens |
| Logging | `log/slog` JSON with request ID | to add | One log line per request that can be found from an error response |
| Configuration | varlock (`.env.schema`, generated `env/env.go`) + committed `.env.example` | in repo | Typed, validated environment with sensitive-value handling; `.env.example` documents it (NFR-04) |
| Object storage (F-11) | `aws-sdk-go-v2` (`service/s3` and `s3.PresignClient`), RustFS `1.0.7` in the compose stack, an AWS S3 bucket in production | to add | One S3 API for both environments; presigned URLs keep file bytes off the API (ADR-0002). `feature/s3/manager` is deprecated and `transfermanager` is 0.x, so neither is used. |
| Tests | Go `testing` + `gotestsum`; `testcontainers-go` (PostgreSQL) for integration; `gofakes3` (in-process S3) for the storage adapter; Vitest via `vp test` + Testing Library + MSW (web); Playwright (E2E) | partly in repo | Covers all three test levels the course requires |
| Packaging | Docker (API image already builds in CI); the image also contains the web build | in repo | Lab 5 |

### 1.3 Module responsibilities

| Module | Responsibility | Does **not** |
| --- | --- | --- |
| React SPA (`apps/web`) | Screens, navigation, quick client-side checks from the generated schemas, showing server errors. Shows decision and edit buttons from the claim's `allowedActions`. | Decide permissions or re-derive business rules. |
| Middleware | Cross-origin protection, session → current user (401), OpenAPI request validation (400), route-level permission (403, Section 5.1). | Check ownership or assignment, which depend on claim data. |
| Handlers | Implement the generated strict server interface: map typed requests to module calls, and module errors to problem responses (Section 4.2). | Contain business rules. |
| Claim module (`internal/claim`) | **All claim business rules** (BR-01 to BR-17): the claim policy, the lifecycle, totals, the conditional update, history, transactions. | Know about HTTP or Gin. |
| Auth module (`internal/auth`) | Login, logout, sessions, bans, route-level permissions. | Know about claims. |
| `internal/db` (sqlc) | Generated queries and row types. | Make permission decisions. |
| `internal/storage` | The S3 adapter: presign PUT and GET, head and read the first bytes of an object, delete, list, and head bucket (F-11). | Know about claims, Receipt File rules, or who may see a file. |
| S3 bucket | Receipt File bytes, private, keyed `receipts/<uuidv7>`. CORS lets the web origin `PUT` and `GET`. | Hold any record of which claim a file belongs to; that is in PostgreSQL. |
| PostgreSQL | Durable storage. Constraints and triggers are the last line of defence: FKs, CHECKs, unique indexes, the append-only history trigger. | – |

**Where rules are enforced:** in the claim module, backed up by database constraints. Tests call the claim module's interface (Section 3.1) against a real PostgreSQL, so they cover how the rules are called, not only the rules alone.

### 1.4 Code organization

Both apps are organized **by feature**: code for one feature lives in one folder, and only a small, named set of shared packages sits outside the features. A sprint's feature contract then maps to one folder per app. Both apps follow the same three rules:

1. **One direction.** Entry/routes → features → shared. Shared code never imports a feature.
2. **No imports between features**, except the `auth` feature, which every feature may use for the current user and permission checks. When two features need the same code, it moves to shared. Features are combined only at the entry point (API) or in a route file (web).
3. **Lint enforces the rules**, so a review does not have to find a broken boundary.

The API and the web split features differently. The API splits by domain aggregate: everything that changes a claim's status goes through one claim module (Section 3), so review and finance actions live in `claim`. The web splits by user journey, so each screen group (Section 7.1) gets a feature.

#### 1.4.1 API (`apps/api`)

This follows the Go guide [Organizing a Go module](https://go.dev/doc/modules/layout), section "Server project": server logic goes in `internal/`, grouped by responsibility. There is only one binary (`migrate` is a subcommand of it), so the existing root `main` package stays and no `cmd/` folder is added.

```
apps/api/
  main.go  server.go  config.go   # entry: startup, config, router; composes the feature handlers
  migrate.go  seed.go  seed_load.go  sweep.go   # `migrate`, `seed`, `seed-load`, and `sweep` subcommands of the one binary
  validate.go                      # request validation middleware: JSON body checked against the contract schema (400 with field errors)
  openapi.yaml                     # the API contract (Section 4)
  oapi-codegen.yaml  sqlc.yaml     # code generation config
  env/                             # varlock-generated (excluded from coverage)
  internal/
    api/                           # shared, generated by oapi-codegen: types + StrictServerInterface
    db/                            # shared, generated by sqlc from every feature's queries.sql
    auth/                          # feature: login/logout/me, sessions, bans, route permissions
      handler.go                   #   implements the auth operations of the strict server interface
      service.go                   #   login, logout, session lookup, ban check
      middleware.go                #   Session: session cookie → current user in the request context
      access.go                    #   statement, roles, RequirePermission, OperationPermissions (Section 5.1)
      queries.sql  *_test.go
    claim/                         # feature: claims, line items, lifecycle, review and finance actions
      handler.go                   #   implements the claim operations
      service.go                   #   the claim module interface (Section 3.1)
      policy.go                    #   who may see a claim and which actions they may take (Section 5.2)
      lifecycle.go                 #   transition table
      rules.go                     #   BR field checks
      receipt.go                   #   Receipt Files: create, complete, link, delete, download, sweep (Section 3.4)
      model.go  queries.sql  *_test.go
    report/                        # feature: decision summaries (Sprint 3)
      handler.go  service.go  queries.sql  *_test.go
    daterange/                     # shared: the Asia/Bangkok `from`/`to` date range of the All Claims filters and the summary (Sprint 3)
    httperr/                       # shared: RFC 9457 problem responses and request IDs (Section 4.2)
    system/                        # shared: /health and /ready (Section 4.3)
    money/                         # shared: the money.Satang type (Section 2.3)
    storage/                       # shared: the S3 adapter over aws-sdk-go-v2 (F-11, Section 3.4)
    pgtest/                        # shared, tests only: one PostgreSQL container per package, a fresh database per test
  migrations/                      # goose SQL migrations (one schema for all features)
```

- **Inside a feature, files split by responsibility.** `handler.go` maps the generated request and response types. `service.go` is the feature's interface and owns transactions. `policy.go`, `lifecycle.go`, and `rules.go` are pure functions behind it.
- **Each feature owns its queries, and one package holds the generated code.** sqlc reads every feature's `queries.sql` and generates a single `internal/db` package. Enum types (claim status, action, category) exist once, so features need no conversion code. A feature may read any table in its own queries. For example, `report` reads `claims` directly instead of importing `claim`.
- **One generated server interface.** `oapi-codegen` generates `api.StrictServerInterface` from `openapi.yaml`. `server.go` builds one struct that embeds `auth.Handler`, `claim.Handler`, and `report.Handler`, so each feature implements only its own operations, and the compiler reports any operation that is missing.
- **Names.** Packages are short singular nouns. Following [Package names](https://go.dev/blog/package-names), there are no `util`, `common`, or `models` packages, and type names do not repeat the package name (`claim.Service`, not `claim.ClaimService`).
- **Enforcement.** `depguard` rules in `.golangci.yml`:
  - `internal/report/**` must not import `internal/claim`, and the reverse.
  - Shared packages (`api`, `daterange`, `db`, `httperr`, `money`, `pgtest`, `storage`) must not import a feature.
- **Receipt Files live in `claim`.** Whether a member may open or change a Receipt File depends on its claim's visibility and status (Section 5.2), and `depguard` forbids a separate feature from importing `claim`. The bucket client is a shared package because readiness (`system`) also checks the bucket.
  - Only `_test.go` files may import `pgtest`.

#### 1.4.2 Web (`apps/web`)

This follows the `features/` model in [bulletproof-react](https://github.com/alan2207/bulletproof-react/blob/master/docs/project-structure.md). The difference: routes stay in `src/routes/`, where TanStack Router's file-based routing generates the route tree. Bulletproof uses `src/app/routes/` instead.

```
apps/web/src/
  main.tsx  router.tsx  astryx-themes.ts  # entry, router + QueryClient setup (router context holds both), generated theme module
  routes/                              # thin route files (Section 7.1): path, beforeLoad guard, loader,
                                       #   search params; render feature components and nothing else
  features/
    auth/                              # login form, useCurrentUser, can(resource, action)
    claims/                            # claims list, claim form + line-item rows, claim detail
      components/claim-form.tsx
      __test__/claim-form.test.tsx     #   tests for this feature (Vitest runs only __test__ folders)
    reviews/                           # decision panel, approve/reject/return dialogs
    finance/                           # final decisions (Sprint 3)
    reports/                           # decision summary (Sprint 3)
  components/ui/                       # shared: buttons, dialog, table, skeleton, toast
  components/                          # shared, claim-aware but feature-free: StatusBadge, Money, DateText
  lib/
    api/                               # shared: generated client, query options, Zod schemas (@hey-api/openapi-ts)
    problem.ts                         # shared: reads RFC 9457 errors into field messages
    form.tsx                           # shared: TanStack Form hook with Astryx field components
    format.ts                          # shared: money and date formatting (Asia/Bangkok)
  __test__/                            # app-level tests (router, routes) and shared test helpers:
                                       #   render helpers, MSW handlers, seed users
```

- **Folders inside a feature are optional.** Add `components/`, `hooks/`, or `api/` only when the feature needs them. Most data access uses the generated query options from `#/lib/api` directly, so a feature's `api/` folder holds only custom hooks.
- **Route files combine features.** For example, `_auth/claims/$claimId.tsx` renders `ClaimDetail` from `claims` and, when `allowedActions` has a decision, `DecisionPanel` from `reviews`. Neither feature imports the other.
- **Imports.** Use the existing `#/` alias, for example `#/features/claims/components/claim-form`. The current `import/no-relative-parent-imports` rule already blocks `../`. Import files directly. There are no barrel `index.ts` files, because they slow down Vite and hide where code comes from.
- **Tests live in `__test__/` folders.** Each feature or shared folder that has tests gets its own `__test__/` folder, for example `features/claims/__test__/claim-form.test.tsx`. Tests for routes and the router, and the shared test helpers, go in `src/__test__/`. `apps/web/vite.config.ts` already sets this: Vitest runs only `src/**/__test__/**/*.test.{ts,tsx}`, and coverage (100% required) excludes `__test__` folders, so test helpers do not count as app code. The Go API keeps Go's own rule instead (Section 10).
- **Enforcement.** `no-restricted-imports` overrides in `apps/web/check.config.ts`:
  - Each `features/<name>/**` may import only `#/features/auth/**` and shared paths.
  - `components/**` and `lib/**` must not import `#/features/**` or `#/routes/**`.

## 2. Data model

### 2.1 Entity–relationship diagram

```mermaid
erDiagram
    USER ||--o{ SESSION : has
    USER ||--|| ACCOUNT : "signs in with (credential)"
    USER ||--o{ CLAIM : submits
    USER ||--o{ CLAIM : "is assigned reviewer of"
    USER |o--o{ USER : "is assigned reviewer of (reporting line)"
    CLAIM ||--|{ LINE_ITEM : contains
    LINE_ITEM |o--o{ RECEIPT_FILE : "is proved by"
    USER ||--o{ RECEIPT_FILE : uploads
    CLAIM ||--o{ CLAIM_HISTORY : records
    USER ||--o{ CLAIM_HISTORY : "acts in"
```

### 2.2 Entities

All `UUID` primary keys are **UUIDv7**, generated in Go with `uuid.NewV7()`.

The auth tables keep three concerns apart:

- `user` holds the member's identity, role, and reporting line.
- `account` holds how the member signs in. Only username + password is used.
- `session` holds the sessions the member is logged in with.

`user` is a reserved word in PostgreSQL, so queries quote it as `"user"`. There is no email verification or password reset (EX-04), and no account administration (EX-08), so the schema has no tables or fields for them.

**`user`**

| Field | Type | Constraints |
| --- | --- | --- |
| id | UUID | PK |
| name | VARCHAR(100) | NOT NULL. Display name, e.g. "Anan K." |
| email | VARCHAR(255) | UNIQUE, NOT NULL. Seed users get `<username>@claimflow.test`; it is not used for login. |
| email_verified | BOOLEAN | NOT NULL, default false. Always true in seed data; it is not checked (EX-04). |
| image | TEXT | NULL. Not used in the UI yet. |
| username | VARCHAR(50) | UNIQUE, NOT NULL. Stored lowercase; the login identifier (FR-01). |
| display_username | VARCHAR(50) | NOT NULL. Username as the user typed it. |
| role | VARCHAR(20) | NOT NULL, default `requester`. CHECK `role IN ('requester', 'reviewer', 'finance')`. Exactly one role per account (D-02). Its permissions are in Section 5.1. |
| banned | BOOLEAN | NOT NULL, default false. A banned user cannot log in and has no valid sessions. This is how an account is deactivated. |
| ban_reason | VARCHAR(200) | NULL |
| ban_expires | TIMESTAMPTZ | NULL means the ban does not expire. When it is in the past, the user is treated as not banned. |
| position | VARCHAR(50) | Display label only, e.g. "Section Head A" |
| reviewer_id | UUID | FK → user.id. The member's **assigned reviewer** (reporting line). NULL means the member cannot create claims (BR-15); in seed data only `finance` users have NULL. CHECK `reviewer_id <> id` (AS-03). |
| created_at, updated_at | TIMESTAMPTZ | NOT NULL |

**`account`** – one row per sign-in method. Only the `credential` provider (username + password) is used.

| Field | Type | Constraints |
| --- | --- | --- |
| id | UUID | PK |
| user_id | UUID | FK → user.id, ON DELETE CASCADE, NOT NULL |
| provider_id | VARCHAR(50) | NOT NULL. Always `credential`. |
| account_id | VARCHAR(255) | NOT NULL. For `credential`, the user's `id`. UNIQUE `(provider_id, account_id)`. |
| password | VARCHAR(100) | bcrypt hash, cost 12 (NFR-01). NOT NULL for `credential` rows (CHECK). |
| created_at, updated_at | TIMESTAMPTZ | NOT NULL |

**`session`**

| Field | Type | Constraints |
| --- | --- | --- |
| id | UUID | PK |
| user_id | UUID | FK → user.id, ON DELETE CASCADE, NOT NULL |
| token | BYTEA | UNIQUE, NOT NULL. **SHA-256 of the cookie token**; the raw token is never stored, so a database leak exposes no usable sessions. |
| expires_at | TIMESTAMPTZ | NOT NULL, created_at + 8 h (NFR-03) |
| ip_address | VARCHAR(45) | NULL. Client IP at login, for audit. |
| user_agent | TEXT | NULL. Client user agent at login, for audit. |
| created_at, updated_at | TIMESTAMPTZ | NOT NULL |

**`claims`**

| Field | Type | Constraints |
| --- | --- | --- |
| id | UUID | PK |
| claim_no | VARCHAR(20) | UNIQUE, NOT NULL. Set at creation from the sequence `claim_no_seq`: `CLM-<year of creation in Asia/Bangkok>-<nextval, 6 digits>`, e.g. `CLM-2026-000123`. The sequence never resets, so numbers are unique even across years. |
| requester_id | UUID | FK → user.id, NOT NULL. The **owner**. |
| assigned_reviewer_id | UUID | FK → user.id. NULL while `DRAFT`. Copied from `user.reviewer_id` at **first** submission and never changed afterwards (BR-17). CHECK `assigned_reviewer_id <> requester_id` (BR-01). CHECK `status = 'DRAFT' OR assigned_reviewer_id IS NOT NULL`. |
| title | VARCHAR(120) | NOT NULL |
| purpose | TEXT | NOT NULL, 1–1000 chars |
| status | ENUM `DRAFT`, `SUBMITTED`, `RETURNED`, `PENDING_FINANCE`, `APPROVED`, `REJECTED`, `WITHDRAWN` | NOT NULL |
| total_satang | BIGINT | NOT NULL, CHECK `> 0 AND <= 5000000` (BR-06). Calculated on the server (BR-08). |
| return_count | SMALLINT | NOT NULL, default 0, CHECK `BETWEEN 0 AND 3` (BR-14) |
| version | INTEGER | NOT NULL, default 0. Incremented on every change; the client sends it back in `If-Match` (Section 3.3, BR-13). |
| first_submitted_at | TIMESTAMPTZ | NULL while `DRAFT`. Set once at first submission; used for the 90-day window (BR-07). |
| submitted_at | TIMESTAMPTZ | Latest submission time. NULL while `DRAFT`. |
| decided_at | TIMESTAMPTZ | NULL until `APPROVED` or `REJECTED` |
| created_at, updated_at | TIMESTAMPTZ | NOT NULL |

Indexes: `(requester_id, status)` and `(assigned_reviewer_id, status)`.

**`line_items`**

| Field | Type | Constraints |
| --- | --- | --- |
| id | UUID | PK |
| claim_id | UUID | FK → claims.id, ON DELETE CASCADE |
| line_no | SMALLINT | NOT NULL, UNIQUE per claim, 1–20 (BR-05) |
| category | ENUM `TRAVEL`, `ACCOMMODATION`, `MEALS`, `SUPPLIES`, `TRAINING`, `OTHER` | NOT NULL (BR-12) |
| description | VARCHAR(200) | NOT NULL |
| expense_date | DATE | NOT NULL. The date window (BR-07) is checked by the claim module on every submit/resubmit. |
| amount_satang | BIGINT | NOT NULL, CHECK `> 0 AND <= 2000000` (BR-05). CHECK `category <> 'MEALS' OR amount_satang <= 150000` (BR-12). |
| receipt_ref | VARCHAR(50) | NOT NULL (D-09). Still required when the line item has Receipt Files. |

**`receipt_files`** (F-11) – one row per Receipt File. The bytes are in the S3 bucket under `object_key`.

| Field | Type | Constraints |
| --- | --- | --- |
| id | UUID | PK |
| object_key | VARCHAR(64) | UNIQUE, NOT NULL. `receipts/<id>`; it never contains user input. |
| uploader_id | UUID | FK → user.id, NOT NULL. Only the uploader can complete, attach, or delete the file. |
| filename | VARCHAR(255) | NOT NULL. The original file name, shown in the UI and sent in the download's `Content-Disposition`. |
| content_type | VARCHAR(50) | NOT NULL. CHECK `IN ('image/jpeg', 'image/png', 'image/webp', 'application/pdf')` (BR-18). |
| size_bytes | INTEGER | NOT NULL. CHECK `BETWEEN 1 AND 10485760` (BR-18). |
| state | ENUM `UPLOADING`, `READY` | NOT NULL, default `UPLOADING`. `READY` after the complete check (Section 3.4). |
| line_item_id | UUID | FK → line_items.id, ON DELETE SET NULL, NULL while unattached. Only a `READY` file can be attached: CHECK `line_item_id IS NULL OR state = 'READY'`. |
| created_at | TIMESTAMPTZ | NOT NULL. The sweep removes unattached files older than 24 hours. |

Indexes: `(line_item_id)` and `(uploader_id, created_at)`. The limit of 5 Receipt Files per line item (BR-18) is checked by the claim module inside the claim transaction, not by a constraint.

**`claim_history`** – append-only (BR-11)

| Field | Type | Constraints |
| --- | --- | --- |
| id | BIGINT | PK, `GENERATED ALWAYS AS IDENTITY`. Gives the order of entries. |
| claim_id | UUID | FK → claims.id, NOT NULL (no cascade) |
| actor_id | UUID | FK → user.id, NOT NULL |
| action | ENUM `SUBMIT`, `APPROVE`, `REJECT`, `RETURN`, `RESUBMIT`, `WITHDRAW`, `FINAL_APPROVE`, `FINAL_REJECT` | NOT NULL |
| from_status | claim status ENUM | NULL for a claim submitted directly (no draft) |
| to_status | claim status ENUM | NOT NULL |
| comment | VARCHAR(500) | NULL. Required (10–500 non-blank chars) for REJECT, RETURN, and FINAL_REJECT (BR-04), checked by the claim module when the entry is written. Entries written under the earlier 5-character minimum are kept as stored and not checked again (BR-11). |
| created_at | TIMESTAMPTZ | NOT NULL, default now() |

- A `BEFORE UPDATE OR DELETE` trigger on `claim_history` raises an error, so no code path can change history (BR-11). An integration test checks this.
- Drafts have no history; history starts at the first `SUBMIT`.
- Deleting a draft (BR-10) deletes the claim and its line items. A claim with history can never be deleted, because the trigger and the FK block it.
- Deleting a draft also deletes its `receipt_files` rows in the same transaction, and their objects after the commit (Section 3.4). Edits replace line items, so the claim module moves each kept Receipt File to the new line item row in the same transaction.

### 2.3 Money

- Amounts are **integer satang** (1 THB = 100 satang) in `BIGINT` columns. In Go they use the type `money.Satang` (an `int64`), and floats are never used.
- `money.Satang` implements JSON marshalling as a decimal **string** with exactly 2 places, e.g. `"1250.00"`. Decoding rejects anything else, including `"1.005"` (BR-05), so no handler parses money. A sqlc override maps the `*_satang` columns to `money.Satang`.
- The contract describes money as `type: string, pattern: '^\d{1,6}\.\d{2}$'`, so the generated web schema checks the format before sending.
- Limits are named constants in `internal/claim/rules.go`: item 20,000.00 (BR-05), Meals item 1,500.00 (BR-12), total 50,000.00 (BR-06), high-value threshold 10,000.00 (BR-09). They are fixed by the SRS, so they are not configuration.
- The claim module recomputes `total_satang = Σ line_items.amount_satang` on every create or edit and ignores any client total (BR-08).

### 2.4 Time

- The claim and auth modules take the current time from an injected `clock` (`func() time.Time`), never from `time.Now()` directly. Tests use a fixed clock.
- "Today" means the calendar date in **Asia/Bangkok** (AS-06). BR-07: `expense_date <= today` and `expense_date >= date(first_submitted_at in Asia/Bangkok) − 90 days`. On first submission, `first_submitted_at` is the current time.
- Timestamps are stored as `TIMESTAMPTZ` (UTC) and converted to Asia/Bangkok only for dates and display.

### 2.5 Migrations and seed data

| Command (in `apps/api`) | Purpose |
| --- | --- |
| `bun run deps:up` (repo root) | `docker compose up -d --wait db rustfs rustfs-init`: PostgreSQL 17 and RustFS with named volumes, and the one-shot bucket and CORS setup, for local development. It replaces `db:up`. |
| `bun run migrate up\|down\|status` | Runs `api migrate`, a subcommand of the API binary. goose applies the SQL files in `migrations/`, which are embedded in the binary, under a PostgreSQL advisory lock. The API never migrates at startup. |
| `bun run migrate:new <name> sql` | Adds a goose SQL migration file |
| `bun run generate` | Regenerate `env/env.go` (varlock), `internal/api` (oapi-codegen), and `internal/db` (sqlc); generated code is committed, and CI fails if regenerating changes it |
| `bun run seed` | Load the base demo dataset. Idempotent: upserts by username/claim_no. |
| `bun run seed:load` | Runs `api seed-load`: adds 240 load claims `CLM-LOAD-0001` to `CLM-LOAD-0240` for NFR-07 (F-10 §1.8). Idempotent by `claim_no`. Needs the base seed users. |
| `bun run sweep` | Runs `api sweep`: deletes unattached Receipt Files older than 24 hours, objects with no `receipt_files` row, and rows whose object is missing (FR-17, Section 3.4). Safe to run again. Run by hand; it is not scheduled. |
| `bun run perf -- --base-url <url> --rounds 20` (repo root) | NFR-07 timed run, `scripts/nfr07-timed-run.ts`: prints the response times of the main requests as a Markdown table and exits non-zero when the overall p95 is 1 s or more (F-10 §1.9). Needs `SEED_USER_PASSWORD` in the environment and the load data. |

In `apps/web`, `bun run generate` regenerates `src/lib/api` from `../api/openapi.yaml`, with the same CI check. The root `bun run generate` runs both.

**Base seed accounts.** Each seed user gets a `user` row and a `credential` `account` row. All share the password in `SEED_USER_PASSWORD` (sensitive, from the environment, never committed – NFR-04). They form the reporting line `Staff → Section Head → Department Head → Finance Officer`:

| Username | Role | Position | Assigned reviewer | Used to test |
| --- | --- | --- | --- | --- |
| `req.anan` | `requester` | Staff, Section A | `rev.malee` | Normal journey |
| `req.ploy` | `requester` | Staff, Section A | `rev.malee` | Cross-requester access (another user's claim) |
| `req.niran` | `requester` | Staff, Section B | `rev.somchai` | Unrelated reviewer access |
| `rev.malee` | `reviewer` | Section Head A | `rev.prasert` | Reviewing; her own claim goes up the line; self-approval attempt |
| `rev.somchai` | `reviewer` | Section Head B | `rev.prasert` | Unrelated reviewer (Section A claims) |
| `rev.prasert` | `reviewer` | Department Head | `fin.wanida` | Top of the reviewer line; his claims go to Finance (ADR-0001) |
| `fin.wanida` | `finance` | Finance Officer | – (NULL) | Reviews the Department Head's claims; cannot create claims (BR-15); final decisions in Sprint 3 |

The seed also creates sample claims in every Sprint 1 status (`SUBMITTED`, `APPROVED`, `REJECTED`), each with history. It creates no Receipt Files. `pgtest` (Go integration tests), MSW handlers (web tests), and Playwright (E2E) use the same seed users.

## 3. Claim module and lifecycle

### 3.1 Interface

Every claim operation goes through one module, `claim.Service`. Its interface is small, and all claim rules sit behind it:

```go
func (s *Service) Get(ctx context.Context, actor Actor, id uuid.UUID) (Claim, error)
func (s *Service) List(ctx context.Context, actor Actor, q Query) (Page, error)
func (s *Service) Create(ctx context.Context, actor Actor, in Input, submit bool) (Claim, error)
func (s *Service) Edit(ctx context.Context, actor Actor, id uuid.UUID, version int, in Input) (Claim, error)
func (s *Service) Delete(ctx context.Context, actor Actor, id uuid.UUID, version int) error
func (s *Service) Act(ctx context.Context, actor Actor, id uuid.UUID, version int, a Action, comment string) (Claim, error)

// Receipt Files (F-11, Section 3.4)
func (s *Service) CreateReceiptFile(ctx context.Context, actor Actor, in ReceiptFileInput) (ReceiptUpload, error)
func (s *Service) CompleteReceiptFile(ctx context.Context, actor Actor, id uuid.UUID) (ReceiptFile, error)
func (s *Service) DeleteReceiptFile(ctx context.Context, actor Actor, id uuid.UUID) error
func (s *Service) ReceiptFileURL(ctx context.Context, actor Actor, id uuid.UUID) (string, error)
func (s *Service) SweepReceiptFiles(ctx context.Context) (SweepResult, error)
```

- `Claim` includes `AllowedActions`, the actions this actor may take on this claim right now (Section 5.2).
- Errors are typed: `ErrNotFound`, `ErrForbidden`, `ErrInvalidTransition{Rule}`, `ErrConflict`, and `ValidationError{Fields}`. The handler maps each one to a problem response (Section 4.2).
- **As built in Sprint 1.** `Query` holds the scope (`ScopeMine` or `ScopeReview`), `Page`, and `PageSize`; F-09 adds its filters. `ErrInvalidTransition` is a plain error, and its rule ID is in `rules.go`. `Create` has no `submit` flag: it always submits, and F-05 adds drafts with its own design. `Edit` and `Delete` come in Sprint 2.
- Each submit-type action (`submit`, `approve`, `reject`, `return`, `withdraw`, `final-approve`, `final-reject`) is a call to `Act`. Every action endpoint becomes a short handler that makes one call.

### 3.2 Lifecycle table

The transition table lives once, in `internal/claim/lifecycle.go`. Who may take each action comes from the claim policy (Section 5.2).

| From | Action | To | Who / guard | Sprint |
| --- | --- | --- | --- | --- |
| – | SUBMIT | SUBMITTED | Owner with an assigned reviewer (BR-15) | 1 |
| SUBMITTED | APPROVE | APPROVED | Assigned reviewer, not the owner | 1 |
| SUBMITTED | REJECT | REJECTED | Assigned reviewer, not the owner; reason required | 1 |
| – | SAVE_DRAFT | DRAFT | Owner with an assigned reviewer (BR-15) | 2 |
| DRAFT | SUBMIT | SUBMITTED | Owner | 2 |
| DRAFT | DELETE | (row removed) | Owner | 2 |
| SUBMITTED | RETURN | RETURNED, `return_count + 1` | Assigned reviewer, not the owner; `return_count < 3` (BR-14) | 2 |
| RETURNED | RESUBMIT | SUBMITTED | Owner; same assigned reviewer (BR-17) | 2 |
| SUBMITTED, RETURNED | WITHDRAW | WITHDRAWN | Owner (BR-16) | 2 |
| SUBMITTED | APPROVE | PENDING_FINANCE | Assigned reviewer whose role is not `finance`; total > 10,000.00 THB (BR-09) | 3 |
| PENDING_FINANCE | FINAL_APPROVE / FINAL_REJECT | APPROVED / REJECTED | Any `finance` user, not the owner | 3 |

`APPROVE` picks its target: `PENDING_FINANCE` if the total is above the high-value threshold and the approver is not a Finance Officer, otherwise `APPROVED`. Before Sprint 3 it always goes to `APPROVED`. The `submit` endpoint records `SUBMIT` for a draft and `RESUBMIT` for a returned claim.

### 3.3 Inside `Act`, `Edit`, and `Delete`

Each call runs in one transaction (`pgx.BeginFunc`, NFR-06):

1. Load the claim. If it does not exist → `ErrNotFound`. If the policy says the actor may not see it → `ErrForbidden`.
2. If the action is not in the policy's allowed actions for this actor → `ErrForbidden` (BR-01, BR-02, BR-15) or `ErrInvalidTransition` (BR-03, BR-10, BR-14, BR-16), with the rule ID. One exception: when the status does not allow the action and the client's `If-Match` version is not the current one, the result is `ErrConflict` (BR-13). Another decision won since the client loaded the claim (F-04 contract revision 0.4).
3. Run the field rules (BR-04 to BR-07, BR-12) → `ValidationError` with one entry per field.
4. Run `UPDATE claims SET status = $to, version = version + 1, ... WHERE id = $id AND version = $expected`, where `$expected` is the version the **client** sent in `If-Match`.
5. If 0 rows are updated, the claim changed after the client loaded it, or another request won a race → `ErrConflict` (409) and roll back. This covers both concurrent decisions (BR-13) and a decision made on an out-of-date screen.
6. Insert the `claim_history` row and commit.

### 3.4 Receipt Files (F-11)

Receipt File rules live in `internal/claim/receipt.go` behind the methods in Section 3.1. The bucket is reached through a small `Store` interface that `claim` defines and `internal/storage` implements, so `claim` never imports the AWS SDK.

```mermaid
sequenceDiagram
    participant B as Browser
    participant A as API (claim module)
    participant D as PostgreSQL
    participant S as S3 bucket
    B->>A: POST /receipt-files {filename, contentType, size}
    A->>A: type and size checks (BR-18)
    A->>D: insert receipt_files (UPLOADING)
    A-->>B: 201 {id, uploadUrl, uploadHeaders, expiresAt}
    B->>S: PUT uploadUrl (Content-Type, Content-Length signed)
    B->>A: POST /receipt-files/{id}/complete
    A->>S: HeadObject, then GET Range bytes=0-511
    A->>D: state = READY
    A-->>B: 200 ReceiptFile
    B->>A: POST /claims {items: [{..., receiptFileIds}]}
    A->>D: claim, line items, and links in one transaction
```

| Step | Rule |
| --- | --- |
| Create | The declared type must be JPEG, PNG, WebP, or PDF (415) and the size 1 byte to 10 MB (413), with a field message (NFR-08). The API inserts an `UPLOADING` row and returns a presigned `PUT` for `receipts/<id>` with `Content-Type` and `Content-Length` signed and a 5-minute expiry. |
| Complete | Uploader only. `HeadObject` must find the object with the declared size and type. The first 512 bytes must sniff as the declared type. On a match the row becomes `READY`. A missing object → 409 `CONFLICT`; a wrong size or type → 415. On either failure the API deletes the object and the row. |
| Attach | `items[].receiptFileIds` in the claim create, edit, and submit bodies. Each ID must exist, belong to the actor, be `READY`, and not be attached to another line item. A line item ends with at most 5. A failure is a 422 field error `items[i].receiptFileIds` with rule BR-18, and nothing is saved. Links are written in the claim's transaction (NFR-06). Omitting an attached ID from an edit detaches it, and the sweep removes it later. |
| Frozen | A Receipt File attached to a claim that is not `DRAFT` or `RETURNED` cannot be detached or deleted: 409 `INVALID_TRANSITION`, rule BR-03. |
| Delete | Uploader only, and only while the file is unattached or its claim is `DRAFT` or `RETURNED`. The row is deleted in the transaction and the object after the commit. If the object delete fails, the row stays deleted and the sweep removes the object. |
| Download | `GET /receipt-files/{id}/content` → 302 to a presigned `GET` with a 5-minute expiry, `response-content-type` set to the stored type, and `response-content-disposition` set to `inline` with the original filename. Visibility follows Section 5.2. |
| Sweep | Deletes unattached Receipt Files older than 24 hours (row and object), objects under `receipts/` with no row, and rows whose object is missing. |

- **Two endpoints, one config.** The API calls the bucket at `S3_ENDPOINT`. It signs URLs for `S3_PUBLIC_ENDPOINT`, because the signed host must be the one the browser reaches (`localhost:9000` locally, while the API reaches `rustfs:9000` inside compose). Path-style addressing is on only when an endpoint is set. In production both are unset, so the SDK uses the regional AWS endpoint.
- **Checksums.** Both SDK clients set `RequestChecksumCalculation` and `ResponseChecksumValidation` to `WhenRequired`, so RustFS and other S3-compatible stores accept the requests.
- **History.** Attaching or removing a Receipt File writes no history entry of its own. Withdraw, reject, and approve keep the files.

## 4. API contract and conventions

### 4.1 General

- **Contract first.** `apps/api/openapi.yaml` (OpenAPI 3.0.3) is the source of truth for every operation, request, response, and error. A feature contract in `docs/sprints/` adds its operations to this file first. Then both apps regenerate their code from it.
- **Paths:** base path `/api/v1`. Resources are nouns (`/claims`), and state changes use action sub-resources (`POST /claims/{id}/approve`), so each transition has its own operation, permission check, and test.
- **Format:** JSON bodies, `Content-Type: application/json`. Field names are camelCase. Money is a 2-decimal string (Section 2.3). Dates are `YYYY-MM-DD`; timestamps are RFC 3339.
- **Identity:** taken **only** from the session, never from a user ID or role in the request.
- **Lists:** `?page=1&pageSize=20` (max 100) → `{ "data": [...], "page": 1, "pageSize": 20, "total": 57 }`.
- **Single item:** `{ "data": { ... } }`.
- **Versions:** claim responses include `version` and the header `ETag: "<version>"`. Every request that changes an existing claim (`PUT`, `DELETE`, and the actions) must send `If-Match: "<version>"`. A missing header → 428; a version that no longer matches → 409 `CONFLICT`. The SRS asks for a conflict error (BR-13), so a stale version also returns 409, not HTTP's usual 412.

### 4.2 Error format

Errors use **RFC 9457 Problem Details** (`Content-Type: application/problem+json`), produced only by `internal/httperr` and defined once in the contract as the `Problem` schema. Responses never include stack traces, SQL, or Go error strings. Two extension members are added: `code` (stable, for the client) and `errors` (one entry per field). `type` is always `about:blank`, so `title` is the HTTP meaning and `code` is what the client checks. `requestId` is a UUIDv7 that the server makes for every request. It is also sent in the `X-Request-Id` header of every response, and an incoming `X-Request-Id` is ignored.

```json
{
  "type": "about:blank",
  "title": "Some fields are invalid.",
  "status": 422,
  "code": "VALIDATION_FAILED",
  "instance": "/api/v1/claims",
  "requestId": "019a5e2c-7b3d-7c41-9f2e-4b8d1a6c3e70",
  "errors": [
    {
      "field": "items[1].expenseDate",
      "rule": "BR-07",
      "message": "Expense date cannot be in the future."
    }
  ]
}
```

| HTTP | `code` | When |
| --- | --- | --- |
| 400 | `BAD_REQUEST` | Malformed JSON, or the request does not match the contract (wrong types, missing fields, bad money format) |
| 401 | `UNAUTHENTICATED` | No session, or the session is invalid or expired, or the user is banned |
| 403 | `FORBIDDEN` | Wrong role for the route, not the owner, not the assigned reviewer, self-approval (BR-01, BR-02), no assigned reviewer (BR-15), or a cross-origin state change |
| 404 | `NOT_FOUND` | Claim or Receipt File does not exist |
| 409 | `INVALID_TRANSITION` | Action not allowed from the current status (BR-03, BR-10, BR-16), or return limit reached (BR-14). `errors[0].rule` names the rule. |
| 409 | `CONFLICT` | The claim changed since the client loaded it, or a concurrent change won (BR-13). Also: a Receipt File is completed before its object is in the bucket (F-11). |
| 413 | `FILE_TOO_LARGE` | A Receipt File is empty or larger than 10 MB (BR-18) |
| 415 | `UNSUPPORTED_MEDIA_TYPE` | A Receipt File is not JPEG, PNG, WebP, or PDF, or its content does not match its type (BR-18) |
| 422 | `VALIDATION_FAILED` | Business-rule validation failed (BR-04 to BR-07, BR-12, BR-18) |
| 428 | `PRECONDITION_REQUIRED` | A change to an existing claim without `If-Match` |
| 500 | `INTERNAL` | Unexpected error. Details are logged with `slog` only, under the same `requestId`. |

A 403 response never includes the claim's data.

### 4.3 Operations

The contract holds the exact shapes. All paths are under `/api/v1`.

| Method | Path | Purpose | Route permission (5.1) | Sprint |
| --- | --- | --- | --- | --- |
| POST | `/auth/login` | Log in, set session cookie | none (anonymous) | 1 |
| POST | `/auth/logout` | Delete session, clear cookie | session | 1 |
| GET | `/auth/me` | Current user `{ id, username, name, role, permissions }`. `permissions` is the role's map from Section 5.1, so the SPA shows only allowed links. | session | 1 |
| POST | `/claims` | Create and submit a claim (Sprint 2: `?draft=true` saves a draft) | `claim:create` | 1 |
| GET | `/claims?scope=…` | List claims in a scope (Section 5.3), with `status`, `category`, `from`, `to`, `q` filters from Sprint 3 | depends on scope | 1 |
| GET | `/claims/{id}` | Claim detail + line items + history + `allowedActions` | session; policy | 1 |
| POST | `/claims/{id}/approve` | Approve `{ comment? }` | session; policy | 1 |
| POST | `/claims/{id}/reject` | Reject `{ comment }` | session; policy | 1 |
| PUT | `/claims/{id}` | Edit a DRAFT/RETURNED claim | session; policy | 2 |
| DELETE | `/claims/{id}` | Delete a DRAFT claim | session; policy | 2 |
| POST | `/claims/{id}/submit` | Submit a draft or resubmit a returned claim | session; policy | 2 |
| POST | `/claims/{id}/return` | Return `{ comment }` (max 3 per claim) | session; policy | 2 |
| POST | `/claims/{id}/withdraw` | Withdraw a SUBMITTED/RETURNED claim | session; policy | 2 |
| POST | `/claims/{id}/final-approve`, `/final-reject` | Finance final decision | session; policy | 3 |
| GET | `/reports/summary?from=&to=` | Decision summary (Section 4.4) | `report:read` | 3 |
| POST | `/receipt-files` | Start a Receipt File upload `{ filename, contentType, size }` → 201 `{ id, uploadUrl, uploadHeaders, expiresAt }` | `receipt:create` | 3 |
| POST | `/receipt-files/{id}/complete` | Check the stored object and mark the Receipt File `READY` | session; uploader | 3 |
| DELETE | `/receipt-files/{id}` | Delete an unattached or editable Receipt File | session; uploader | 3 |
| GET | `/receipt-files/{id}/content` | 302 to a presigned download URL | session; policy | 3 |
| GET | `/health` | Liveness, no auth, no DB: `{ status, version }` | none | 1 |
| GET | `/ready` | Readiness, no auth: pings the database and runs `HeadBucket` (F-11); 503 if either fails | none | 1 |

**Filters (FR-13, Sprint 3):** `status` and `category` match exactly (a claim matches `category` if any line item has it). `from`/`to` are inclusive Asia/Bangkok dates on the first submission date, or on the creation date for drafts. `q` is a case-insensitive substring match on the title.

### 4.4 Decision summary (FR-14, Sprint 3)

`GET /reports/summary?from=&to=` covers claims whose decision (`decided_at`, Asia/Bangkok date) falls in the range. The scope is the claims assigned to a `reviewer`, or every claim for `finance`.

```json
{
  "data": {
    "countsByStatus": {
      "SUBMITTED": 4,
      "RETURNED": 1,
      "PENDING_FINANCE": 2,
      "APPROVED": 12,
      "REJECTED": 3,
      "WITHDRAWN": 1
    },
    "approvedTotals": [
      { "month": "2026-10", "category": "TRAVEL", "total": "18250.00" }
    ],
    "averageDecisionHours": 26.4
  }
}
```

- `countsByStatus` counts every claim in the scope that is not a draft, by current status. The date range does not apply to this field.
- `approvedTotals` sums line-item amounts of `APPROVED` claims, grouped by the month of `decided_at` and by category.
- `averageDecisionHours` is the mean of `decided_at − first_submitted_at` for `APPROVED` and `REJECTED` claims in the range.

## 5. Roles, ownership, and permissions

Access is checked at two levels:

- **Route permissions (5.1)** depend only on the role. They gate operations that are not about one claim.
- **The claim policy (5.2)** depends on the actor and the claim's data. It decides everything about one claim.

### 5.1 Route permissions

Route permissions are defined in `internal/auth/access.go`. A **statement** lists the resources and actions, and each **role** gets a subset. Middleware `auth.RequirePermission(resource, action)` returns **403** when `user.role` does not grant the action. The router applies the same check per contract operation through `auth.OperationPermissions`, after the request is validated; a feature adds its operation to `operationPermissions` in `apps/api/routes.go`. Code checks permissions, never role names.

```go
var statement = Statement{
    "claim":   {"create", "list"}, // create claims; list own claims
    "review":  {"list"},           // list the review queue
    "finance": {"list"},           // list the finance queue (Sprint 3)
    "report":  {"read"},           // decision summary (Sprint 3)
    "receipt": {"create"},         // start a Receipt File upload (F-11)
}

var roles = map[Role]Permissions{
    RoleRequester: {"claim": {"create", "list"}, "receipt": {"create"}},
    RoleReviewer:  {"claim": {"create", "list"}, "receipt": {"create"}, "review": {"list"}, "report": {"read"}},
    RoleFinance:   {"review": {"list"}, "finance": {"list"}, "report": {"read"}},
}
```

`/auth/me` returns the actor's permissions, so the web shows only allowed navigation. There is no admin UI, so there are no permissions to manage users or sessions (EX-08).

### 5.2 Claim policy

`internal/claim/policy.go` is a pure function: `Allowed(actor, claim) → (visible bool, actions []Action)`. `Get`, `List`, `Act`, `Edit`, and `Delete` all use it, and `Get` returns its actions to the web as `allowedActions`. "Own" means `requester_id = actor.id`. "Assigned" means `assigned_reviewer_id = actor.id`.

**Visible** if any of these is true:

- own;
- assigned (not `DRAFT`, since drafts have no assigned reviewer);
- from Sprint 3, the actor is `finance` and the claim is not `DRAFT` (department-wide view, FR-13, FR-14).

**Allowed actions** (only for a visible claim, and only if the lifecycle table allows the action from the current status):

| Action | Allowed when | Rule |
| --- | --- | --- |
| `edit` | own, status `DRAFT` or `RETURNED` | BR-08, BR-10 |
| `delete` | own, status `DRAFT` | BR-10 |
| `submit` | own, status `DRAFT` or `RETURNED`, owner has an assigned reviewer (drafts only) | BR-15, BR-17 |
| `withdraw` | own, status `SUBMITTED` or `RETURNED` | BR-16 |
| `approve`, `reject` | assigned **and not own**, status `SUBMITTED` | BR-01, BR-02 |
| `return` | as `approve`, and `return_count < 3` | BR-14 |
| `final-approve`, `final-reject` | actor is `finance` **and not own**, status `PENDING_FINANCE` | BR-01, BR-02 |

The "not own" check uses `requester_id` directly, so self-approval is blocked even if the assignment data is wrong (BR-01).

**Receipt Files (F-11).** An attached Receipt File is visible exactly when its claim is visible. An unattached one is visible only to its uploader. Anyone else gets 403 with no data; an unknown ID gets 404. Changing a Receipt File needs the uploader and, if it is attached, the claim's `edit` action.

### 5.3 List scopes and permission matrix

`GET /claims?scope=` selects one of these scopes. Each scope is the visible set from 5.2, narrowed further:

| Scope | Claims | Route permission | Sprint |
| --- | --- | --- | --- |
| `mine` | own claims, all statuses | `claim:list` | 1 |
| `review` | assigned, status `SUBMITTED` (FR-05) | `review:list` | 1 |
| `finance` | status `PENDING_FINANCE` | `finance:list` | 3 |
| `all` | every visible claim, with filters (FR-13) | session | 3 |

| Operation | `requester` | `reviewer` | `finance` | Enforced by |
| --- | --- | --- | --- | --- |
| Login / logout / me | ✓ | ✓ | ✓ | session middleware |
| Create claim / save draft | ✓ | ✓ | ✗ 403 (BR-15) | `claim:create` + claim module: `reviewer_id IS NOT NULL` |
| List own claims (`scope=mine`) | ✓ | ✓ | ✗ 403 | `claim:list` |
| Review queue (`scope=review`) | ✗ 403 | assigned only | assigned only (Department Head's claims) | `review:list` + scope |
| View claim detail | own | own **or** assigned | assigned; any non-draft from S3 | policy |
| Edit / submit / withdraw / delete | own, allowed status | own, allowed status | ✗ 403 | policy |
| Approve / reject / return | ✗ 403 | assigned **and** not own | assigned **and** not own | policy |
| Final approve / reject (S3) | ✗ 403 | ✗ 403 | PENDING_FINANCE, not own | policy |
| Decision summary (S3) | ✗ 403 | assigned claims | whole department | `report:read` + scope |
| Start a Receipt File upload (F-11) | ✓ | ✓ | ✗ 403 | `receipt:create` |
| Complete, attach, or delete a Receipt File (F-11) | uploader; claim `DRAFT`/`RETURNED` | uploader; claim `DRAFT`/`RETURNED` | ✗ 403 | claim module (3.4) |
| Open a Receipt File (F-11) | uploader, or claim visible | uploader, or claim visible | claim visible | policy (5.2) |

**Required access tests (Sprint 1, API integration level):**

- **Route inventory (NFR-02):** a test walks every operation in `openapi.yaml`. Each one except `/auth/login`, `/health`, and `/ready` returns 401 without a session. A new operation is covered without writing a new test.
- `req.anan` logs in with the right password → 200 and a session cookie. With a wrong password, or as an unknown user → 401 with the same generic message.
- `req.ploy` requests `req.anan`'s claim → 403, no claim data in the body.
- `req.anan` calls `/claims/{id}/approve` → 403.
- `rev.somchai` approves `req.anan`'s claim, which is assigned to `rev.malee` → 403 (unrelated reviewer).
- `rev.malee` approves her own claim, which is assigned to `rev.prasert` → 403. With the assignment data corrupted to point to herself, the response is still 403 (BR-01).
- `fin.wanida` creates a claim → 403 (BR-15).
- A banned user logs in → 401 with the generic message; a session created before the ban → 401.
- `fin.wanida` approves `rev.prasert`'s claim → 200 (Finance acting as assigned reviewer).
- A state change with a foreign `Origin` → 403 (cross-origin protection).

## 6. Authentication and sessions

| Aspect | Design |
| --- | --- |
| Credentials | Username + password. The username is lowercased, then the `user` row and its `credential` `account` row are loaded. The password is checked against `account.password` with `bcrypt.CompareHashAndPassword`. |
| Failed login | 401 with the generic message "Invalid username or password." A banned user (`banned = true` and `ban_expires` NULL or in the future) gets the same response, so the response does not show which accounts exist or are banned. To limit timing differences, an unknown username is still compared against a dummy hash. |
| Session creation | 32 bytes from `crypto/rand`, base64url-encoded, becomes the cookie value. Its SHA-256 is stored in `session.token` with `expires_at = now + 8h`, `ip_address`, and `user_agent`. `ip_address` is gin's `ClientIP`, which reads `X-Forwarded-For` behind Cloud Run, so it is an audit hint and never an access check. An empty value is stored as NULL. The 8-hour lifetime is a constant, not configuration (NFR-03). |
| Cookie | `__Host-sid=<token>`; `HttpOnly`; `Secure`; `SameSite=Lax`; `Path=/`; no `Domain`; `Max-Age=28800`. The `__Host-` prefix makes the browser enforce `Secure`, `Path=/`, and no `Domain`. Local development over plain HTTP (`COOKIE_SECURE=false`) uses the name `sid` without `Secure`. |
| Request handling | The `auth.Session` Gin middleware hashes the cookie, loads an unexpired session of a user who is not banned through `auth.Service`, and stores the user in the request context. A request with no such session continues without a user. `auth.RequireSession` then returns 401 for it on every matched route except the public ones, the operations that `openapi.yaml` marks `security: []` (login, health, ready). The check fails closed: a route added outside the contract, or one whose key does not match the Gin route, needs a session. A request that matches no route still gets 404. Handlers read the user with `auth.UserFrom`; route-level permissions (Section 5.1) build on the same value. |
| Current user | `GET /auth/me`. The `_auth` route's `beforeLoad` calls `queryClient.ensureQueryData` for it, puts the user in router context, and redirects anonymous users to `/login?redirect=<path>` before any protected screen renders. |
| Logout | Delete the session row and clear the cookie. Reusing the old token afterwards → 401 (NFR-03). |
| Expiry | Fixed lifetime, no sliding renewal. Expired rows are deleted at login. |
| Ban | Setting `banned = true` (through seed data or a migration; there is no admin UI, EX-08) also deletes all of the user's `session` rows. |
| CSRF | `http.CrossOriginProtection` wraps the router. It rejects non-safe cross-origin browser requests using `Sec-Fetch-Site`, falling back to `Origin`. `SameSite=Lax` is a second layer. |
| Why server-side sessions | Logout takes effect immediately, which a stateless JWT cannot do. |

## 7. UI design

### 7.1 Screens and navigation

TanStack Router file routes live in `apps/web/src/routes/`. Routes under `_auth` require a session (Section 6). Loaders call `queryClient.ensureQueryData` with the generated query options, so data starts loading with navigation instead of after the first render.

| Screen | Route file → path | Roles | Sprint |
| --- | --- | --- | --- |
| Login | `login.tsx` → `/login` | Anonymous | 1 |
| My Claims (list) | `_auth/claims/index.tsx` → `/claims` | Requester, Reviewer | 1 |
| New Claim (form with line-item rows, each with a Receipt File picker from F-11) | `_auth/claims/new.tsx` → `/claims/new` | Requester, Reviewer | 1 |
| Claim Detail (line items with their Receipt Files, total, status, history timeline, action buttons from `allowedActions`) | `_auth/claims/$claimId.tsx` → `/claims/:id` | Anyone the policy lets see the claim | 1 |
| Review Queue | `_auth/reviews.tsx` → `/reviews` | Reviewer, Finance | 1 |
| Edit Claim | `_auth/claims/$claimId.edit.tsx` | Owner (DRAFT/RETURNED) | 2 |
| Finance Queue (final decisions) | `_auth/finance.tsx` → `/finance` | Finance | 3 |
| Summary / Reports | `_auth/reports.tsx` → `/reports` | Reviewer, Finance | 3 |

- **Header:** display name, role, the links that `permissions` allows, and **Log out**.
- **After login:** Requesters go to `/claims`; Reviewers and Finance go to `/reviews`.
- **Finance users:** no "My Claims" or "New Claim" links, because their role has no `claim` permissions (BR-15).
- **Buttons come from the server.** The web shows an action button only if it is in the claim's `allowedActions`. It never re-derives status or ownership rules.

### 7.2 Shared UI behaviour

| State | Behaviour |
| --- | --- |
| Loading | Skeleton rows for lists; disabled button with spinner while submitting (prevents double submit). |
| Empty | Friendly message plus next action, e.g. "No claims yet – Create claim". |
| Validation | Field-level message under each input from the problem's `errors` (NFR-08). The client first checks fields with the generated Zod schemas. |
| 401 | Clear the query cache and redirect to `/login?redirect=<path>`. |
| 403 / 404 | Inline "You don't have access to this claim" / "Claim not found" page, with no data shown. |
| 409 | Toast "This claim was changed by someone else – reloaded", then refetch. |
| 5xx / network error | Inline error with **Try again**, which retries the query. Mutations are not retried automatically. The `requestId` is shown in small text for support. |
| Decisions | Reject/Return open a dialog with a required reason field. Approve asks for confirmation. The Return button shows "Returned 2 / 3" from `returnCount`; it is shown only when `return` is in `allowedActions`. |
| Status badges | Draft (grey), Submitted (blue), Returned (amber), Pending finance (purple), Approved (green), Rejected (red), Withdrawn (grey, struck through). |
| Money / dates | `฿1,250.00`; `31 Oct 2026` in Asia/Bangkok (AS-06). |
| Receipt Files (F-11) | Picking a file starts its upload with `XMLHttpRequest`, because `fetch` reports no upload progress. Each file shows a progress bar, **Cancel** while it runs, and **Remove** once it is done. Type and size are checked in the browser first for quick feedback; the server checks again. Uploaded files stay on the form after a server rule error. On Claim Detail each file opens the download endpoint in a new tab. |

## 8. Configuration and secrets

Configuration is declared in each app's varlock `.env.schema`:

- `bun run codegen` regenerates `apps/api/env/env.go`.
- The API validates its config at startup (`config.go`) and refuses to start if a value is missing or invalid.
- Sensitive values are marked `@sensitive`, so varlock redacts them in logs.
- `.env` files are git-ignored. Each app commits a `.env.example` that lists every variable with a safe example value (NFR-04, NFR-10). The root `.env.example` holds the few values `docker compose up` reads.

| Variable (apps/api) | Example | Required | Sensitive |
| --- | --- | --- | --- |
| `GIN_MODE` | `release` | existing | no |
| `PORT` | `8080` | existing | no |
| `DATABASE_URL` | `postgres://app:<password>@localhost:5432/claimflow` | yes | **yes** |
| `COOKIE_SECURE` | `true` in production | default `true`; `false` only for local HTTP | no |
| `BCRYPT_COST` | `12` | default `12`; must be ≥ 10 (NFR-01). Tests may lower it to 10 for speed. | no |
| `LOG_LEVEL` | `info` | default `info` | no |
| `SEED_USER_PASSWORD` | – | seed only | **yes** |
| `APP_VERSION` | git SHA / release tag, set at build | no | no |
| `WEB_DIR` | `/app/web` in the image; unset in development | no | no |
| `S3_BUCKET` | `claimflow-receipts` | yes (F-11) | no |
| `S3_REGION` | `ap-southeast-1`; any value for RustFS | yes (F-11) | no |
| `S3_ENDPOINT` | `http://localhost:9000` locally, `http://rustfs:9000` in compose; unset in production | no | no |
| `S3_PUBLIC_ENDPOINT` | `http://localhost:9000`; defaults to `S3_ENDPOINT` | no | no |
| `AWS_ACCESS_KEY_ID` | – | yes (F-11) | **yes** |
| `AWS_SECRET_ACCESS_KEY` | – | yes (F-11) | **yes** |

Business limits (amount caps, the high-value threshold, the 8-hour session lifetime, the return limit) come from the SRS, so they are constants in code and not configuration.

`apps/web` keeps its existing `API_URL` for the dev proxy. The web bundle contains no secrets.

On GCP, sensitive values live in **Secret Manager** and are injected into Cloud Run as environment variables (Lab 5). The AWS key pair for the Receipt File bucket is stored the same way (F-11).

## 9. Deployment design (implemented in Lab 5)

```mermaid
flowchart LR
    Dev[Developer] -->|PR| GH[GitHub]
    GH -->|CI: format, lint, generate check,<br/>tests, coverage, vuln, image build| CI{Checks pass?}
    CI -- no --> Block[Merge blocked]
    CI -- yes --> Stg[sprint-N<br/>sprint branch]
    Stg -->|release PR| Main[main]
    Main --> Rel[semantic-release<br/>tag vG.X.Y.Z]
    Rel --> Build[Build image<br/>API + web]
    Build --> AR[Artifact Registry]
    AR --> Mig[goose up<br/>Cloud Run job]
    Mig --> CR[Cloud Run: claimflow]
    CR -- Cloud SQL connector --> SQL[(Cloud SQL<br/>PostgreSQL 17)]
    SM[Secret Manager] -. env vars .-> CR
    CR -- "aws-sdk-go-v2" --> AS3[(AWS S3 bucket<br/>ap-southeast-1)]
```

| Concern | Design |
| --- | --- |
| Image | One image from the root `Dockerfile`, built with the repository root as context. A web stage runs `vp build`, the Go stage builds the API, and the Alpine runtime (varlock, runs as `nobody`, port 8080) contains the binary and the web `dist/` at `/app/web` (`WEB_DIR`). The build argument `APP_VERSION` sets the version. Migrations and the seed are subcommands of the same binary. |
| Web delivery | The Gin router serves `dist/` for every path outside `/api`, falling back to `index.html` for client-side routes. Hashed assets get `Cache-Control: public, max-age=31536000, immutable`, and `index.html` gets `no-cache`. One origin means the session cookie works without CORS. Only the Receipt File bucket has CORS, for the web origin (F-11). |
| Local | `docker compose up --build` runs `postgres:17` (named volume, so data survives restarts – NFR-05), a one-shot `migrate` service (`migrate up`, then `seed`), `rustfs/rustfs:1.0.7` (named volume, `/health` healthcheck, S3 port on loopback), a one-shot `rustfs-init` (`amazon/aws-cli`) that creates the bucket and sets CORS for `http://localhost:3000` and `http://localhost:8080`, and the image as `app`, which waits for `rustfs-init` (NFR-10). |
| Cloud runtime | **Cloud Run** (stateless) + **Cloud SQL for PostgreSQL 17** (durable). Sessions are stored in the DB, so instance restarts do not log users out. The startup probe calls `/ready`, so a revision gets traffic only once it can reach the database. |
| Object storage (F-11) | An AWS S3 bucket in `ap-southeast-1`, created by the idempotent `deploy/aws-storage.sh`: Block Public Access on, SSE-S3, versioning off, CORS for the Cloud Run URL, and a lifecycle rule that aborts incomplete multipart uploads after 1 day. The API uses an IAM user whose policy allows only `s3:GetObject`, `s3:PutObject`, `s3:DeleteObject`, and `s3:ListBucket` on that bucket. The production image contains no storage server. |
| Migrations | `goose up` runs as a Cloud Run job before the new revision gets traffic. The app never changes the schema on startup. |
| Logs | `slog` JSON on stdout with Cloud Logging field names (`severity`, `message`), one line per request with `requestId`, method, route, status, and latency. Sensitive values are never logged. |
| Release identity | Images are tagged with the full git SHA only and deployed by digest; `/health` returns that SHA as `version`. A semantic-release tag `vG.X.Y.Z` (G = generation, from `release.config.js`) maps to its SHA with `git rev-list -n 1 vG.X.Y.Z`. |
| CI gate | The existing `CI` workflow runs format, lint, web tests, Go tests with **100% coverage**, govulncheck, and the image build on every PR. It also checks that the generated code is up to date: it runs `generate` and fails on any diff. GitHub rulesets `main`, `sprint`, and `lab` (Section 9.1) make the `code_quality`, `test`, `api`, `build`, and `image` jobs required checks. `image` builds the root image. |
| Branches | Repo convention: `<type>/<short-kebab-name>` (e.g. `feat/claim-submission`). Each sprint has a **staging branch** (`sprint-1`, `sprint-2`, `sprint-3`). Feature PRs target the current sprint's staging branch and are squash-merged; at the end of the sprint one release PR merges it into `main` with a merge commit, which triggers semantic-release. `CONTRIBUTING.md` documents the flow. |

### 9.1 Branch rulesets

The three rulesets are active on GitHub. The `lab` ruleset protects the Lab staging branches (`lab-5`) with the same rules as `sprint`. `main` has no bypass actor. On `sprint` and `lab`, the repository admin can bypass, so a staging branch can be rebased onto `main` before a release; every other change goes through a pull request.

| Rule | `main` ruleset (`main`) | `sprint` ruleset (`sprint-*`) and `lab` ruleset (`lab-*`) |
| --- | --- | --- |
| Branch deletion | Blocked | Blocked |
| Force push | Blocked | Blocked |
| Pull request | Required, 1 approval | Required, 1 approval |
| Review | Stale approvals dismissed on push | Stale approvals dismissed on push |
| Merge method | Merge commit only | Squash only |
| Linear history | – | Required |
| Required checks | `code_quality`, `test`, `api`, `build`, `image` | `code_quality`, `test`, `api`, `build`, `image` |
| Up to date before merging | Required | Required |
| Bypass | None | Repository admin |

- Squash merges into staging make each PR title one commit subject. The merge commit into `main` keeps those subjects, so semantic-release reads one release signal per feature.
- A PR needs 1 approval, and GitHub never lets authors approve their own PR. A new push dismisses the approval. The required checks and the review are the gate. Last-push approval is off: with two people, it would block every PR that the other person rebased.

## 10. Test strategy (system level)

Tests go through each module's interface. They test behaviour, not internals, so a refactor inside a module does not break them.

| Level | Tool | Covers | Example |
| --- | --- | --- | --- |
| Policy matrix (Go) | `testing`, table-driven, in-process | `claim.Allowed`: every role × ownership/assignment × status → visible and allowed actions | `rev.malee` on her own `SUBMITTED` claim → no `approve` |
| Value types (Go) | `testing`, table-driven | `money.Satang` JSON decoding, clock-based date window | BR-05 boundaries: `"0.00"`, `"0.01"`, `"20000.00"`, `"20000.01"`, `"1.005"`; BR-07 at Bangkok midnight |
| Claim module (Go) | `claim.Service` against `pgtest` PostgreSQL, fixed clock | Every BR through `Create`/`Edit`/`Act`; transactions; history rows; the history trigger | Self-approval → `ErrForbidden`; status and history unchanged |
| HTTP (Go) | `httptest` + full router + `pgtest`; the real `internal/storage` adapter against an in-process `gofakes3` server (F-11) | Contract validation, problem responses, cookies, cross-origin protection, the route inventory and access tests (5.3), Receipt File rules | Anonymous request to every operation → 401 |
| Concurrency (Go) | Two goroutines `Act` on the same claim with the same version | BR-13 | Exactly one success, one `ErrConflict`, one history row |
| Unit/component (web) | Vitest via `vp test`, Testing Library, MSW handlers typed from the generated client | Forms, problem → field messages, `allowedActions` → buttons, navigation by permission | Finance user sees no "New Claim" link |
| E2E | Playwright against a local seeded stack before each release, not in CI (Product DoD item 4) | Critical journeys; the only proof of the real presign, CORS, and RustFS path (F-11) | Requester submits → reviewer approves → requester sees Approved with history |

Test files follow each language's standard layout:

- **Go:** `_test.go` files sit in the same folder as the code they test. The Go toolchain requires this: `go test` finds tests only in the package folder, and only a test file in that folder can test unexported code. Shared test helpers live in `internal/pgtest`. `gofakes3` does not check presigned signatures, so router tests write the object directly in place of the browser's `PUT`. It starts one PostgreSQL container per test package, migrates a template database once, and gives each test a fresh database copied from the template (`CREATE DATABASE ... TEMPLATE`), so tests stay independent without migrating again.
- **Web:** `*.test.ts(x)` files sit in a `__test__/` folder (Section 1.4.2).

Both apps require **100% coverage**: the API through `.testcoverage.yml`, the web through `test.coverage.thresholds` in `apps/web/vite.config.ts`. Only generated code is excluded. For the API that is `env/`, `internal/api/`, and `internal/db/`; for the web it is `src/lib/api/` and `src/routeTree.gen.ts`. The mapping from AC to test IDs is in each feature's STS and in `docs/traceability.md`.

## 11. Design decisions

| ID | Decision | Alternatives considered | Reason |
| --- | --- | --- | --- |
| SD-01 | Modular monolith: one Go API and one SPA | Microservices | Enough for the scope; a simple deployment. |
| SD-02 | Server-side sessions in the DB with an HttpOnly cookie | JWT in localStorage | Logout takes effect immediately (NFR-03), and scripts cannot read the token. |
| SD-03 | Action endpoints (`/approve`, `/reject`, …), each one call to `claim.Service.Act` | Generic `PATCH status` | Each transition has its own operation and test; the rules still live in one module. |
| SD-04 | Copy the assigned reviewer onto the claim at first submission | Look it up live every time | Later reporting-line changes do not move claims already in review (BR-17). |
| SD-05 | Optimistic locking with `version`, sent by the client in `If-Match` | `SELECT … FOR UPDATE`; version read inside the transaction | One check covers concurrent decisions and out-of-date screens, with a clear 409 (BR-13). |
| SD-06 | Integer satang in DB/Go (`money.Satang`), decimal strings in JSON | NUMERIC + a decimal library; floats | No rounding errors, no extra dependency, and no handler parses money. |
| SD-07 | Reporting line stored as `user.reviewer_id` | Org-unit/position tables | Covers the 4-level line with one column. |
| SD-08 | Finance Officer reviews the Department Head; that approval is final | – | ADR-0001 |
| SD-09 | `sqlc` + `pgx` instead of an ORM, one generated `internal/db` package | GORM, ent; one sqlc package per feature | Explicit SQL for the conditional update; enum types generated once. |
| SD-10 | Separate `user`, `account`, and `session` tables; one role column on `user` | One `users` table with the password hash; a separate auth service | Credentials and sessions stay apart from identity, so a new sign-in method or a password change does not touch `user`. One role per account matches D-02. |
| SD-11 | Two access levels: role permissions for routes, a claim policy for each claim | Role permissions for every action; checks spread over handlers | Assignment and ownership already decide claim actions. One policy answers both "may I?" and "which buttons?". |
| SD-12 | Feature-based folders in both apps, with boundaries enforced by lint (Section 1.4) | Layer folders at the top level (`handlers/`, `services/`, `components/`, `hooks/`) | A sprint feature changes one folder per app, and the boundaries stay visible as the app grows. |
| SD-13 | Contract first: a hand-written OpenAPI file, with code generated for both apps | Code-first (Huma, swag); hand-written types in both apps | The spec is the feature contract the course asks for, and drift becomes a compile or CI error. |
| SD-14 | RFC 9457 problem details with `code` and `errors` extensions | Custom error envelope | A standard format that HTTP tools understand, defined once in the contract. |
| SD-15 | One image: the API serves the web build | Cloud Storage + CDN; a separate web service | One origin for the cookie, one deploy, no CORS for the app. Since F-11 only the Receipt File bucket has CORS (SD-17). |
| SD-16 | Server-computed `allowedActions` on every claim | Web derives buttons from role and status | Rules live only in the claim policy; the web cannot drift from them. |
| SD-17 | Receipt Files in S3; the browser moves the bytes with presigned URLs; downloads are a 302 to a presigned `GET` | Go CDK through the API (#61); GCS XML API with HMAC keys; keyless web identity federation | ADR-0002. The API stays small and fast, and one S3 API serves RustFS and AWS. |
| SD-18 | Receipt File rules in `claim`; the bucket client in the shared `internal/storage` | A separate `receipt` feature | Who may see or change a file depends on its claim, and `depguard` forbids importing `claim` from another feature (1.4.1). |

## 12. Revision history

| Version | Date | Change | Author |
| --- | --- | --- | --- |
| 0.1 | 2026-10-31 | Initial system design draft | 1005, 1045 |
| 0.2 | 2026-10-31 | Aligned with SRS v0.2: reporting-line seed data, FINANCE role from Sprint 1, WITHDRAWN, `return_count`, `first_submitted_at`, draft delete | 1005, 1045 |
| 0.3 | 2026-10-31 | Adopted the repo stack: Go/Gin, React 19 + TanStack Router, Bun monorepo, varlock, existing CI; integer-satang money; sqlc/pgx/goose; 100% Go coverage | 1005, 1045 |
| 0.4 | 2026-11-02 | Repo renamed to `claimflow` (#19); sprint staging branches `staging-1`–`staging-3` in the deployment flow | 1005, 1045 |
| 0.5 | 2026-11-04 | Auth tables `user`, `account`, `session` with roles, bans, and username fields; lowercase role values; permission-based access control (5.1); SD-10, SD-11 | 1005, 1045 |
| 0.6 | 2026-11-04 | Feature-based code organization for API and web (1.4); sqlc output per feature; SD-12 | 1005, 1045 |
| 0.7 | 2026-11-04 | Architecture review: claim module interface (3.1) and claim policy with `allowedActions` (5.2); list scopes (5.3); OpenAPI contract with generated server and client; RFC 9457 errors; `If-Match` versions; `money.Satang`; injected clock and Asia/Bangkok dates; one sqlc package; UUIDv7; `__Host-` cookie and `CrossOriginProtection`; `/ready`; web served by the API. Fixed SRS gaps: BR-12 Meals cap, BR-11 history trigger, fixed 8 h session and high-value threshold, claim number format, FR-13 filters, FR-14 summary shape, route-inventory and login tests, 5xx UI state. SD-13 to SD-16. Documented the existing branch rulesets (9.1). Web tests live in `__test__` folders, matching `apps/web/vite.config.ts`. | 1005, 1045 |
| 0.8 | 2026-11-04 | Auth design (2.2, 5.1, 6, SD-10) written as this system's own design, with no references to outside frameworks | 1005, 1045 |
| 0.9 | 2026-11-07 | Sprint branches renamed to `sprint-1`–`sprint-3` and the ruleset to `sprint` (#7); both rulesets require 1 approval, dismiss stale approvals, and block last-pusher approval (9.1) | 1005, 1045 |
| 0.10 | 2026-11-07 | F-01/01 (#24) decisions: HeroUI v3 (1.2); `system` package and per-test template databases in `pgtest` (1.4, 10); `db:up` at the root, `migrate up\|down\|status` subcommand with embedded migrations and an advisory lock (2.5); problem `type` is `about:blank`, and `requestId` is a server-made UUIDv7 also sent in `X-Request-Id` (4.2) | 1005, 1045 |
| 0.11 | 2026-11-13 | Web UI: Astryx with the Neutral theme replaces Tailwind and HeroUI; TanStack Form in use through `#/lib/form` (1.2, 1.4.2) | 1005, 1045 |
| 0.12 | 2026-11-15 | Sprint 1 release. Records the changes that feature PRs made: `OperationPermissions` and the per-operation check (1.4.1, 5.1, PR #59); `validate.go` (1.4.1, PR #60); the stale-version exception in 3.3 (PR #65, F-04 contract 0.4). Adds what Sprint 1 built differently from 3.1 and 8 (the `List` methods, string actions, empty session audit fields), tracked in #67 | 1005 |
| 0.16 | 2026-11-22 | E2E (10): Playwright runs against a local seeded stack before each release, not in CI, per Product DoD 0.3 item 4 (#66) | beambeambeam |
| 0.17 | 2026-11-22 | Sprint 1 follow-ups (#67, #70): §3.1 "As built" note says `Create` has no `submit` flag; session creation stores `ip_address` from gin's `ClientIP` and the `User-Agent` | beambeambeam |
| 0.19 | 2026-11-27 | F-10 review (#119): the new shared package `internal/daterange` holds the Bangkok date range that `claim` (F-09) and `report` (F-10) both read (1.4.1); `httperr.ErrUnauthenticated` replaces the copy in each of `auth`, `claim`, and `report` | 1005 |
| 0.18 | 2026-11-24 | Lab 7 change L7/01 (#99), SRS 0.3: the BR-04 reason minimum in `claim_history.comment` (2.2) is 10 characters, not 5. No schema change: the column stays `VARCHAR(500)` NULL, and existing history rows are not checked again | 1005 |
| 0.20 | 2026-11-27 | F-10/02: `seed-load` and `seed_load.go` (1.4.1), the `seed:load` and root `perf` commands (2.5) | 1005 |
| 0.15 | 2026-11-19 | Release identity (9): images carry the SHA only; a version maps to its SHA (#54) | 1005 |
| 0.14 | 2026-11-18 | 9.1 matches the live rulesets: admin bypass on `sprint` and `lab`, up to date before merging on all three, no last-push approval rule (#53) | 1005 |
| 0.13 | 2026-11-17 | Lab 5 (#53): the root `Dockerfile` and `WEB_DIR` (8, 9); the `migrate` compose service (9); `image` replaces `api_image` and is a required check in the `main`, `sprint`, and `lab` rulesets; the `lab` ruleset is documented (9, 9.1) | 1005 |
| 0.21 | 2026-11-28 | F-11 Receipt Files (Sprint 3), SRS 0.4 (ADR-0002): storage path (1.1); `aws-sdk-go-v2`, RustFS, `gofakes3` (1.2, 10); `internal/storage` and `receipt.go` (1.3, 1.4.1); `receipt_files` (2.1, 2.2); `deps:up` replaces `db:up`, and `sweep` (2.5); Receipt File methods and rules (3.1, 3.4); 413, 415 (4.2); the `/receipt-files` operations and the bucket check in `/ready` (4.3); `receipt:create` (5.1); Receipt File visibility (5.2, 5.3); upload UI (7.1, 7.2); `S3_*` and `AWS_*` (8); RustFS in compose and the AWS bucket (9); SD-15 amended; SD-17, SD-18 | beambeambeam |
