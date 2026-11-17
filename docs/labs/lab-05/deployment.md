# Lab 5 deployment

How ClaimFlow is built, configured, and run, locally and on GCP. The handout is [`lab-5-cloud-deployment.md`](../../project-specifications/lab-5-cloud-deployment.md); the design is [SDS](../../system-sds.md) §8, §9, and §9.1.

This first version covers the image, the local stack, and CI (#53). The GCP resources and the release procedure come with the deploy pipeline (#54).

## Stack and changes from the reference

The Student Guide's reference kit is a Node.js app with Prisma, built as two images (`client`, `server`). ClaimFlow keeps the reference GCP services and changes the app side:

| Concern | Reference kit | ClaimFlow | Why |
| --- | --- | --- | --- |
| App | Node.js + Prisma | Go (Gin) API, React web app built with Vite+ | The term project's stack |
| Image | `client` (nginx) + `server` | One image: the API serves the web build (SD-15) | One origin for the session cookie, no CORS, one deploy |
| Migrations | `prisma migrate deploy` | `api migrate up`, goose migrations embedded in the binary | Same binary, same image |
| Seed | `npm run prisma:seed` | `api seed`: 7 demo users and sample claims, safe to run again | `TestSeedIsIdempotent` |
| Config | `.env` read by the app | varlock validates every variable against `.env.schema` before the API starts | An invalid value stops the API at startup |
| Database | Cloud SQL for PostgreSQL 16 | Cloud SQL for PostgreSQL 17 | Matches `postgres:17` locally and in tests |
| Staging branch | `staging` | `lab-5` for Lab 5, `sprint-N` during sprints | Repo convention (CONTRIBUTING.md) |

## Environment variables

The API reads its settings only from the environment. [`apps/api/.env.schema`](../../../apps/api/.env.schema) declares each one, and [`apps/api/.env.example`](../../../apps/api/.env.example) lists them with placeholder values. No real value is in Git.

| Variable | Purpose | Secret | Local value | On GCP |
| --- | --- | --- | --- | --- |
| `DATABASE_URL` | PostgreSQL connection string | **Yes** | `apps/api/.env`, or set by `docker-compose.yml` for the local `db` | Secret Manager (#54) |
| `SEED_USER_PASSWORD` | Password of the 7 seeded demo users; read only by `api seed` | **Yes** | Root `.env` (compose) or `apps/api/.env` | Secret Manager (#54) |
| `PORT` | Listen port | No | `8080` (default) | Cloud Run sets it |
| `GIN_MODE` | `debug`, `release`, or `test` | No | `debug` for `bun run dev`, `release` otherwise | Default `release` |
| `COOKIE_SECURE` | `Secure` and the `__Host-` prefix on the session cookie | No | `false` (plain HTTP) | Default `true` (HTTPS) |
| `BCRYPT_COST` | bcrypt cost, at least 10 (NFR-01) | No | `12` (default) | Default `12` |
| `LOG_LEVEL` | `debug`, `info`, `warn`, or `error` | No | `info` (default) | Default `info` |
| `APP_VERSION` | Version that `/api/v1/health` returns | No | `dev` | The full commit SHA, set at image build |
| `WEB_DIR` | Folder of the web build that the API serves outside `/api` | No | Unset: the Vite dev server serves the web app | `/app/web`, set in the image |

The web app has one variable, `API_URL`, which only the dev server's proxy reads ([`apps/web/.env.example`](../../../apps/web/.env.example)). The bundle holds no secrets.

The root [`.env.example`](../../../.env.example) holds what `docker compose` reads: `SEED_USER_PASSWORD`, the host ports `DB_PORT` and `APP_PORT`, and `APP_VERSION`. Compose passes every variable in `.env` to the `migrate` and `app` containers, so any other API variable (for example `LOG_LEVEL`) can be added there. `DATABASE_URL` and `COOKIE_SECURE` are fixed in `docker-compose.yml`.

## Run the whole stack locally (C-03)

Needs Docker with Compose. From a fresh clone:

```sh
cp .env.example .env          # then set SEED_USER_PASSWORD to a local value
docker compose up --build -d  # db, then migrate (migrate up + seed), then app
curl -s localhost:8080/api/v1/health   # {"status":"ok","version":"dev"}
curl -s localhost:8080/api/v1/ready    # 200 once the database answers
```

Open <http://localhost:8080> and sign in as a seeded user, for example `req.anan`, with the password from `.env`.

| Service | Image | What it does |
| --- | --- | --- |
| `db` | `postgres:17` | Data in the named volume `db`, so it survives restarts and rebuilds |
| `migrate` | `claimflow:local` | One shot: `api migrate up`, then `api seed`. `app` starts only when it succeeds |
| `app` | `claimflow:local` | The API on port 8080, serving the web build |

Data survives `docker compose restart app` and `docker compose up --build`, because only the `db` volume holds it. `docker compose down -v` deletes it.

## Build, test, migrate, seed

From the repository root unless noted. bun runs the scripts; the API scripts run through varlock.

| Task | Command |
| --- | --- |
| Install | `bun install` |
| Build the image | `docker build --build-arg APP_VERSION=$(git rev-parse HEAD) -t claimflow .` |
| Build without Docker | `bun run build` (web `dist/` and the API binary) |
| Web tests | `bun run --cwd apps/web test:coverage` |
| API tests | `bun run --cwd apps/api test:coverage` (needs Docker for PostgreSQL; 100% coverage required) |
| Lint and format | `bun run check` |
| Database only, for the dev servers | `bun run db:up` |
| Migrate | `bun run --cwd apps/api migrate up` |
| Seed | `bun run --cwd apps/api seed` (needs `SEED_USER_PASSWORD`) |
| Dev servers | `bun run dev` (API on 8080, web on 3000 with the `/api` proxy) |

In the image the same commands are `api migrate up` and `api seed`, run through `varlock run`.

## CI

The `CI` workflow ([`.github/workflows/ci.yml`](../../../.github/workflows/ci.yml)) runs on every pull request and on pushes to `main`:

| Job | What it checks |
| --- | --- |
| `code_quality` | Format and lint |
| `test` | Web tests with 100% coverage |
| `api` | Generated code is current, modules are tidy, golangci-lint, Go tests with 100% coverage, govulncheck |
| `build` | `bun run build` |
| `image` | Builds the root image with `APP_VERSION` set to the commit SHA |

All five are required checks in the `main` and `sprint` rulesets (SDS §9.1), so a pull request with a failing check cannot merge.

## GCP and release procedure

Filled in by #54: project, region, and resource names; the deploy workflow; and how to find the commit that is running.
