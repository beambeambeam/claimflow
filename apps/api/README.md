# api

Go API built with [Gin](https://gin-gonic.com). It logs JSON with `slog` and shuts down gracefully on `SIGINT` or `SIGTERM`.

## Configuration

[Varlock](https://varlock.dev/integrations/go/) validates these variables against `.env.schema` and passes them to the API. Run the API with `varlock run` (`bun run dev`, `bun run migrate`, `bun run seed`, and the Docker image do this). An invalid value stops it before the server starts. `config.go` reads the result with the generated `env` package and checks `BCRYPT_COST` again.

Copy `.env.example` to `.env` for local values. Never commit `.env`.

| Variable | Default | Description |
| --- | --- | --- |
| `PORT` | `8080` | Listen port (0–65535) |
| `GIN_MODE` | `release` | `debug`, `release`, or `test` (`bun run dev` sets `debug`) |
| `DATABASE_URL` | – (required) | PostgreSQL connection string |
| `COOKIE_SECURE` | `true` | Adds `Secure` and the `__Host-` prefix to the session cookie. Set `false` only for local HTTP |
| `BCRYPT_COST` | `12` | bcrypt cost for password hashes, 10–31 |
| `LOG_LEVEL` | `info` | `debug`, `info`, `warn`, or `error` |
| `APP_VERSION` | `dev` | Release tag or git SHA that `/health` returns |
| `SEED_USER_PASSWORD` | unset | Password for the seven demo users. Read only by `api seed`, which fails when it is unset. Never commit a value |

## Database

PostgreSQL 17 runs in Docker Compose. From the repo root:

```sh
bun run db:up      # start PostgreSQL on localhost:5432 (user, password, and database: claimflow)
bun run --cwd apps/api migrate up
```

`bun run migrate <up|down|status>` runs `api migrate`. `up` applies every pending migration, `down` rolls back the latest one, and `status` lists them. The API never migrates at startup. Migrations are SQL files in `migrations/`, embedded in the binary and applied by [goose](https://pressly.github.io/goose/) under a PostgreSQL advisory lock. Add one with `bun run migrate:new <name> sql`.

`bun run seed` runs `api seed` after `migrate up`. It creates or updates the seven SDS §2.5 demo users (`fin.wanida`, `rev.prasert`, `rev.malee`, `rev.somchai`, `req.anan`, `req.ploy`, `req.niran`) and stores a bcrypt hash of `SEED_USER_PASSWORD` for each. Set `SEED_USER_PASSWORD` in `apps/api/.env` (local only, never committed) or in the environment. Without it, `seed` fails and writes nothing. It also adds the six sample claims of F-04 §4 with their line items and history. They have fixed claim numbers (`CLM-2026-900001` to `CLM-2026-900006`), so a second run updates the same users and adds no claim twice.

`bun run db:down` stops the database and keeps its volume.

## Code generation

`bun run generate` (or the root `bun run generate`) regenerates every committed generated file:

| Source | Tool | Output |
| --- | --- | --- |
| `.env.schema` | varlock | `env/env.go` |
| `openapi.yaml` | [oapi-codegen](https://github.com/oapi-codegen/oapi-codegen) (Gin strict server) | `internal/api/api.gen.go` |
| `migrations/*.sql` + `internal/<feature>/*.sql` | [sqlc](https://sqlc.dev) (pgx/v5) | `internal/db/*.gen.go` |

CI runs the root `bun run generate` and fails when the result differs from the commit. Coverage excludes the generated packages.

## Errors

Every 4xx and 5xx response is an [RFC 9457](https://www.rfc-editor.org/rfc/rfc9457) `application/problem+json` body with `type` (`about:blank`), `title`, `status`, `code`, `instance`, and `requestId`. A 500 never contains the internal error; the log line with the same `requestId` does. Every response carries the request ID in `X-Request-Id`. The server always makes a new UUIDv7 and ignores an incoming header.

## Endpoints

The contract is `openapi.yaml`. Paths start at `/api/v1`.

| Method | Path | Response |
| --- | --- | --- |
| `GET` | `/api/v1/health` | 200 `{"status":"ok","version":"<APP_VERSION>"}`. Does not use the database |
| `GET` | `/api/v1/ready` | 200 `{"status":"ok"}` when the database answers, else 503 `UNAVAILABLE` |

## Tests

Integration tests use `internal/pgtest`. It starts one PostgreSQL container per test package with [testcontainers-go](https://golang.testcontainers.org), migrates a template database once, and gives each test a fresh copy. Docker must run.

## Scripts

Run these in `apps/api`. Root scripts call them for you.

| Script | Description |
| --- | --- |
| `bun run dev` | Live reload with [air](https://github.com/air-verse/air), under `varlock run` |
| `bun run build` | Build `dist/api` |
| `bun run generate` | Regenerate env, server, and query code |
| `bun run migrate <up\|down\|status>` | Run migrations against `DATABASE_URL` |
| `bun run migrate:new <name> sql` | Add a migration file |
| `bun run seed` | Create or update the seven demo users and add the sample claims under `varlock run`; needs `SEED_USER_PASSWORD` |
| `bun run test` | Run tests with the race detector |
| `bun run test:coverage` | Run tests and require 100% coverage ([go-test-coverage](https://github.com/vladopajic/go-test-coverage)) |
| `bun run lint` / `bun run lint:fix` | Lint with [golangci-lint](https://golangci-lint.run) |
| `bun run format` / `bun run format:check` | Format with gofumpt and goimports |
| `bun run check` | Format check, lint, and module tidy check |
| `bun run tidy` | Tidy `go.mod`, `tools.mod`, and `golangci-lint.mod` |
| `bun run vuln` | Scan for known vulnerabilities with govulncheck |

## Tooling

Dev tools are pinned with `go tool` in separate module files, so they never enter the app's `go.mod`:

- `golangci-lint.mod`: golangci-lint (isolated, as its docs recommend). `bin/golangci-lint` runs it for editors.
- `tools.mod`: air, gotestsum, go-test-coverage, govulncheck, oapi-codegen, sqlc, goose.

Update a tool with `go get -tool -modfile=<file> <module>@<version>`.

## Docker

```sh
docker build --tag claimflow-api .
docker run --rm -p 8080:8080 -e DATABASE_URL=postgres://... claimflow-api
```

The image runs the API with `varlock run` on Alpine. The varlock binary needs `libstdc++`, so the image can't be distroless.
