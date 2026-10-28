# karupan

Bun workspace monorepo. Apps live in `apps/*`.

| Path       | Description                                    |
| ---------- | ---------------------------------------------- |
| `apps/web` | Web app (React, TanStack Router, Tailwind CSS) |
| `apps/api` | API (Go, Gin)                                  |

## Getting started

Requires [Bun](https://bun.sh), Node.js (version in `.node-version`), and Go (version in `apps/api/go.mod`).

```sh
bun install
bun run dev
```

## Scripts

Run these at the repository root:

| Script                                    | Description                  |
| ----------------------------------------- | ---------------------------- |
| `bun run dev`                             | Start every app dev server   |
| `bun run build`                           | Build every app              |
| `bun run test` / `bun run test:coverage`  | Test every app               |
| `bun run lint` / `bun run lint:fix`       | Lint the whole repository    |
| `bun run format` / `bun run format:check` | Format the whole repository  |
| `bun run check`                           | Format, lint, and type-check |

Repository-wide lint, format, and pre-commit rules live in the root `vite.config.ts`; app-specific rules live in each app (for example `apps/web/check.config.ts`). Share dependency versions through `workspaces.catalog` in the root `package.json` and reference them with `"catalog:"`.

See [`CONTRIBUTING.md`](CONTRIBUTING.md) for the workflow.

## Releases

The `main` branch CI workflow creates a release after formatting, linting, tests, and the production build pass. semantic-release reads Conventional Commit messages, creates a `vG.X.Y.Z` Git tag, and publishes release notes in GitHub Releases. `G` is a fixed generation number set in `release.config.js`; commits only change `X.Y.Z`. There is no version commit, and nothing is published to npm.

Use these commit prefixes to select the next version:

- `fix: correct the sign-in redirect` creates a patch release (`v1.1.0.0` → `v1.1.0.1`).
- `feat: add course search` creates a minor release (`v1.1.0.1` → `v1.1.1.0`).
- `feat!: change the course data format` creates a major release (`v1.1.1.0` → `v1.2.0.0`).
- `chore: update dependencies` does not create a release.

The release job uses the workflow `GITHUB_TOKEN` with `contents: write` to push the tag and publish the GitHub Release. If a tag ruleset protects `v*` tags, add GitHub Actions to its bypass list.
