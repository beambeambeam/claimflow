# ClaimFlow

Bun workspace monorepo. Apps live in `apps/*`.

| Path       | Description                              |
| ---------- | ---------------------------------------- |
| `apps/web` | Web app (React, TanStack Router, Astryx) |
| `apps/api` | API (Go, Gin)                            |

## Getting started

Requires [Bun](https://bun.sh), Node.js (version in `.node-version`), Go (version in `apps/api/go.mod`), and Docker.

```sh
bun install
cp apps/api/.env.example apps/api/.env
bun run db:up
bun run --cwd apps/api migrate up
bun run dev
```

The web app runs on http://localhost:3000 and the API on http://localhost:8080. The web dev server proxies `/api` requests to the API, so the browser calls the same origin and needs no CORS setup. To use another API, copy `apps/web/.env.example` to `apps/web/.env` and change `API_URL`.

## Demo data

A new database needs its tables and the demo data before the app has anything to show. Set `SEED_USER_PASSWORD` in `apps/api/.env` (local only, never committed) to a password of your choice, then run:

```sh
bun run db:up                      # start PostgreSQL
bun run --cwd apps/api migrate up  # create the tables
bun run --cwd apps/api seed        # add the demo users and sample claims
```

`seed` adds seven demo accounts, all with the password from `SEED_USER_PASSWORD`, and six sample claims with their history. It runs again without harm: users are updated in place, and a sample claim is found by its claim number (`CLM-2026-9000xx`) and added only once.

| Username                   | Role                       | Reviewer      |
| -------------------------- | -------------------------- | ------------- |
| `req.anan`, `req.ploy`     | Requester                  | `rev.malee`   |
| `req.niran`                | Requester                  | `rev.somchai` |
| `rev.malee`, `rev.somchai` | Reviewer                   | `rev.prasert` |
| `rev.prasert`              | Reviewer (Department Head) | `fin.wanida`  |
| `fin.wanida`               | Finance Officer            | –             |

Sample claims: two `Submitted`, one `Approved`, and one `Rejected` (with a reason) from `req.anan`, `req.ploy`, and `req.niran` to their reviewers, plus a `Submitted` claim each from `rev.malee` and `rev.prasert` (40,000.00) for the next reviewer up.

## End-to-end tests

The Playwright specs in `apps/web/e2e` run against a running stack. They do not start servers. Seed the database as above, start `bun run dev`, install the browser once, and run the specs with the seed password:

```sh
bunx playwright install chromium
SEED_USER_PASSWORD=<the seed password> bun run --cwd apps/web e2e
```

Set `E2E_BASE_URL` when the web app is not on http://localhost:3000.

## Restart check (NFR-05, F04-TEST-19)

Claims, line items, and history must survive a restart of the API and the database. To check it:

1. Seed the database and run the app. As `req.anan`, submit a claim. As `rev.malee`, approve it.
2. Note what `req.anan` sees: My Claims and the claim with its items, total, and history (or save `GET /api/v1/claims` and each `GET /api/v1/claims/{id}`).
3. Stop the API, then stop the database with `docker compose stop db`. Start it again with `bun run db:up`, which keeps the named volume, then start the API.
4. Log in as `req.anan` again and read the same pages. They must match step 2 exactly.

`bun run db:down` keeps the volume too. `docker compose down -v` deletes it and all data.

## Scripts

Run these at the repository root:

| Script | Description |
| --- | --- |
| `bun run dev` | Start every app dev server |
| `bun run db:up` / `bun run db:down` | Start or stop local PostgreSQL |
| `bun run generate` | Regenerate env, API server, queries, and web client |
| `bun run build` | Build every app |
| `bun run test` / `bun run test:coverage` | Test every app |
| `bun run lint` / `bun run lint:fix` | Lint the whole repository |
| `bun run format` / `bun run format:check` | Format the whole repository |
| `bun run check` | Format, lint, and type-check |

Repository-wide lint, format, and pre-commit rules live in the root `vite.config.ts`; app-specific rules live in each app (for example `apps/web/check.config.ts`). Share dependency versions through `workspaces.catalog` in the root `package.json` and reference them with `"catalog:"`.

See [`CONTRIBUTING.md`](CONTRIBUTING.md) for the workflow.

## Releases

The `main` branch CI workflow creates a release after formatting, linting, tests, and the production build pass. semantic-release reads Conventional Commit messages, creates a `vG.X.Y.Z` Git tag, and publishes release notes in GitHub Releases. `G` is a fixed generation number set in `release.config.js`; commits only change `X.Y.Z`. There is no version commit, and nothing is published to npm. Versioning starts from the `v0.0.0.0` tag on the initial commit.

Use these commit prefixes to select the next version:

- `fix: correct the sign-in redirect` creates a patch release (`v0.0.1.0` → `v0.0.1.1`).
- `feat: add course search` creates a minor release (`v0.0.1.1` → `v0.0.2.0`).
- `feat!: change the course data format` creates a major release (`v0.0.2.0` → `v0.1.0.0`).
- `chore: update dependencies` does not create a release.

The release job uses the workflow `GITHUB_TOKEN` with `contents: write` to push the tag and publish the GitHub Release. If a tag ruleset protects `v*` tags, add GitHub Actions to its bypass list.
