# Monorepo

- Use Bun workspaces. Apps live in `apps/*`. Run `bun install` at the root.
- `apps/api` is a Go app. Its `package.json` only maps workspace scripts to `go` commands. Run `gofmt -w .`, `go vet ./...`, and `go test ./...` in `apps/api` after Go changes.
- Share dependency versions through the `workspaces.catalog` in the root `package.json`.
- Lint, format, and staged-file rules live in the root `vite.config.ts`. Do not add `lint` or `fmt` blocks to app configs; use root `overrides` instead.

# Linting and formatting

- After making code changes, run `bun run lint:fix`, then run `bun run format` at the root.
- Before finishing, run `bun run lint --deny-warnings --format=agent` at the root.
