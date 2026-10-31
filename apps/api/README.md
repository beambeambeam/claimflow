# api

Go API built with [Gin](https://gin-gonic.com). It logs JSON with `slog` and shuts down gracefully on `SIGINT` or `SIGTERM`.

## Configuration

[Varlock](https://varlock.dev/integrations/go/) validates these variables against `.env.schema` and passes them to the API. Run the API with `varlock run` (`bun run dev` and the Docker image do this). An invalid value stops it before the server starts. `config.go` reads the result with the generated `env` package.

After you change `.env.schema`, run `bun run codegen` to regenerate `env/env.go`.

| Variable | Default | Description |
| --- | --- | --- |
| `PORT` | `8080` | Listen port (0–65535) |
| `GIN_MODE` | `release` | `debug`, `release`, or `test` (`bun run dev` sets `debug`) |

## Endpoints

| Method | Path      | Response          |
| ------ | --------- | ----------------- |
| `GET`  | `/health` | `{"status":"ok"}` |

## Scripts

Run these in `apps/api`. Root scripts call them for you.

| Script | Description |
| --- | --- |
| `bun run dev` | Live reload with [air](https://github.com/air-verse/air), under `varlock run` |
| `bun run build` | Build `dist/api` |
| `bun run codegen` | Regenerate `env/env.go` from `.env.schema` |
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
- `tools.mod`: air, gotestsum, go-test-coverage, govulncheck.

Update a tool with `go get -tool -modfile=<file> <module>@<version>`.

## Docker

```sh
docker build --tag claimflow-api .
docker run --rm -p 8080:8080 claimflow-api
```

The image runs the API with `varlock run` on Alpine. The varlock binary needs `libstdc++`, so the image can't be distroless.
