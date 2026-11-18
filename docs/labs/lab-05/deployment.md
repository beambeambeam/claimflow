# Lab 5 deployment

How ClaimFlow is built, configured, and run, locally and on GCP. The handout is [`lab-5-cloud-deployment.md`](../../project-specifications/lab-5-cloud-deployment.md); the design is [SDS](../../system-sds.md) §8, §9, and §9.1.

It covers the image, the local stack, and CI (#53), and the GCP resources (#54). The release procedure comes with the deploy workflow (#54).

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
| `DATABASE_URL` | PostgreSQL connection string | **Yes** | `apps/api/.env`, or set by `docker-compose.yml` for the local `db` | Secret Manager (`claimflow-database-url`) |
| `SEED_USER_PASSWORD` | Password of the 7 seeded demo users; read only by `api seed` | **Yes** | Root `.env` (compose) or `apps/api/.env` | Secret Manager (`claimflow-seed-password`) |
| `PORT` | Listen port | No | `8080` (default) | Cloud Run sets it |
| `GIN_MODE` | `debug`, `release`, or `test` | No | `debug` for `bun run dev`, `release` otherwise | Default `release` |
| `COOKIE_SECURE` | `Secure` and the `__Host-` prefix on the session cookie | No | `false` (plain HTTP) | Default `true` (HTTPS) |
| `BCRYPT_COST` | bcrypt cost, at least 10 (NFR-01) | No | `12` (default) | Default `12` |
| `LOG_LEVEL` | `debug`, `info`, `warn`, or `error` | No | `info` (default) | Default `info` |
| `APP_VERSION` | Version that `/api/v1/health` returns | No | `dev`, or the SHA passed from the shell to `docker compose` | The full commit SHA, set at image build |
| `WEB_DIR` | Folder of the web build that the API serves outside `/api` | No | Unset: the Vite dev server serves the web app | `/app/web`, set in the image |

The web app has one variable, `API_URL`, which only the dev server's proxy reads ([`apps/web/.env.example`](../../../apps/web/.env.example)). The bundle holds no secrets.

The root [`.env.example`](../../../.env.example) holds what `docker compose` reads: `SEED_USER_PASSWORD` and the host ports `DB_PORT` and `APP_PORT`. `APP_VERSION` comes from the shell instead, because a value in `.env` would override the version built into the image. Compose passes every variable in `.env` to the `migrate` and `app` containers, so any other API variable (for example `LOG_LEVEL`) can be added there. `DATABASE_URL` and `COOKIE_SECURE` are fixed in `docker-compose.yml`.

## Run the whole stack locally (C-03)

Needs Docker with Compose. From a fresh clone:

```sh
cp .env.example .env          # then set SEED_USER_PASSWORD to a local value
APP_VERSION=$(git rev-parse HEAD) docker compose up --build -d  # db, then migrate (migrate up + seed), then app
curl -s localhost:8080/api/v1/health   # {"status":"ok","version":"<the commit SHA>"}
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

All five are required checks in the `main`, `sprint`, and `lab` rulesets (SDS §9.1), so a pull request with a failing check cannot merge. The `lab` ruleset protects `lab-5`.

## GCP resources

Project `cpe334-claimflow`, region `asia-southeast1`. [`deploy/setup.sh`](../../../deploy/setup.sh) creates every resource once and deploys the first release (Student Guide Episode 5). The deploy workflow (#54) uses the same names.

| Resource | Name | Settings |
| --- | --- | --- |
| Artifact Registry | `claimflow` | Docker images, tagged with the full commit SHA |
| Cloud SQL | `claimflow-pg`, database `claimflow`, user `claimflow` | PostgreSQL 17, `db-f1-micro` (Enterprise edition), 10 GB SSD, storage auto-increase off, no automated backups |
| Secret Manager | `claimflow-database-url` | `DATABASE_URL` through the Cloud SQL unix socket, with a generated password |
| Secret Manager | `claimflow-seed-password` | `SEED_USER_PASSWORD`, generated |
| Service account | `claimflow-runtime` | Runs the service and the job. Cloud SQL Client on the project, Secret Accessor on the two secrets only |
| Cloud Run job | `claimflow-migrate` | `api migrate up`, then `api seed`, before every service update |
| Cloud Run service | `claimflow` | Public, port 8080, 0 to 2 instances, 512 MiB. Only `DATABASE_URL` is mounted |

Passwords are generated by the script and exist only in Secret Manager. Nobody types or prints them.

### First setup (Episode 5)

In Cloud Shell, on a clean clone at the latest `main` commit: the script refuses any other commit, because only an accepted release deploys (R-07). Cloud SQL is billed for every hour it exists, so check that the budget alert exists first.

```sh
git clone https://github.com/beambeambeam/claimflow.git && cd claimflow
export PROJECT_ID=cpe334-claimflow
gcloud config set project "$PROJECT_ID"
DRY_RUN=1 bash deploy/setup.sh   # prints the plan and creates nothing
bash deploy/setup.sh             # 10 to 20 minutes; safe to run again after a failure
```

Then check the service:

```sh
URL=$(gcloud run services describe claimflow --region asia-southeast1 --format='value(status.url)')
curl -s "$URL/api/v1/health"   # {"status":"ok","version":"<the commit SHA>"}
curl -s "$URL/api/v1/ready"    # 200 once the service reaches Cloud SQL
```

The script builds the image with Docker in Cloud Shell, the same BuildKit build as CI and `docker compose`, instead of Cloud Build. The root `Dockerfile` uses BuildKit cache mounts. It deploys by digest, so the running image is the one built from that commit.

Last, open `$URL` in a browser and sign in as a demo user, for example `req.anan` (Episode 5 is done when the public URL opens and you can sign in, R-03). Read the password privately with `gcloud secrets versions access latest --secret=claimflow-seed-password`. Never show it on screen, in a screenshot, or in a chat.

## Release procedure

Filled in by #54: the deploy workflow, rollback, and how to find the commit that is running.
