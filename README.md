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

Lint, format, and pre-commit rules live in the root `vite.config.ts`. Share dependency versions through `workspaces.catalog` in the root `package.json` and reference them with `"catalog:"`.

See [`CONTRIBUTING.md`](CONTRIBUTING.md) for the workflow.

## Releases

The `main` branch CI workflow creates a release after formatting, linting, tests, and the production build pass. Releases use Conventional Commit messages and are recorded as `vX.Y.Z` Git tags with release notes in GitHub Releases. semantic-release updates the root `package.json` and commits the version before tagging; this private app does not publish to npm.

Use these commit prefixes to select the next SemVer level:

- `fix: correct the sign-in redirect` creates a patch release.
- `feat: add course search` creates a minor release.
- `feat!: change the course data format` creates a major release.
- `chore: update dependencies` does not create a release.

The release job uses the workflow `GITHUB_TOKEN` with `contents: write` to push the version commit and tag and to publish the GitHub Release. If a branch ruleset or tag ruleset blocks pushes to `main` or `v*` tags, add GitHub Actions to its bypass list.
