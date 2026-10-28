# Monorepo

- Use Bun workspaces. Apps live in `apps/*`. Run `bun install` at the root.
- `apps/api` is a Go app. Its `package.json` only maps workspace scripts to `go` commands. Dev tools are pinned with `go tool`: golangci-lint in `golangci-lint.mod`, the rest in `tools.mod`. Never add them to `go.mod`.
- After Go changes, run `bun run format`, `bun run lint:fix`, `bun run test:coverage` (100% required; mark untestable blocks with `// coverage-ignore`), and `bun run vuln` in `apps/api`. After dependency changes, run `bun run tidy`.
- Share dependency versions through the `workspaces.catalog` in the root `package.json`.
- Repository-wide lint, format, and staged-file rules live in the root `vite.config.ts`. App-specific rules live in the app (for example `apps/web/check.config.ts`) as overrides scoped to the app path, and the root config imports them. Do not add `lint` or `fmt` blocks to app `vite.config.ts` files.

# Linting and formatting

- After making code changes, run `bun run lint:fix`, then run `bun run format` at the root.
- Before finishing, run `bun run check` at the root.

## Agent skills

### Issue tracker

Issues live as local markdown files under `.scratch/<feature>/`. See `docs/agents/issue-tracker.md`.

### Domain docs

Single-context: one root `GLOSSARY.md` and `docs/adr/`. See `docs/agents/domain.md`.
