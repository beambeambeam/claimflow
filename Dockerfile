# syntax=docker/dockerfile:1

# One image (SD-15): the API binary and the web build it serves on one origin.
# Build it from the repository root: docker build --build-arg APP_VERSION=$(git rev-parse HEAD) .

FROM node:26-bookworm-slim AS web
WORKDIR /src
RUN npm install --global bun@1.4.4
COPY package.json bun.lock tsconfig.json ./
COPY apps/web/package.json apps/web/
COPY apps/api/package.json apps/api/
# The root `prepare` script installs Git hooks, and there is no Git here.
RUN --mount=type=cache,target=/root/.bun/install/cache bun install --frozen-lockfile --ignore-scripts
COPY apps/web apps/web
RUN bun run --cwd apps/web build

FROM ghcr.io/dmno-dev/varlock:1.24.0 AS varlock

FROM golang:1.27.4 AS api
WORKDIR /src
COPY apps/api/go.mod apps/api/go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download
COPY apps/api/*.go ./
COPY apps/api/env ./env
COPY apps/api/internal ./internal
COPY apps/api/migrations ./migrations
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/api .

# The varlock binary needs libstdc++ and CA certificates, so the image uses Alpine instead of distroless.
FROM alpine:3.24
RUN apk add --no-cache ca-certificates libstdc++
COPY --from=varlock /usr/local/bin/varlock /usr/local/bin/varlock
WORKDIR /app
COPY apps/api/.env.schema ./
COPY --from=api /out/api ./api
COPY --from=web /src/apps/web/dist ./web
# The full commit SHA of the build. /api/v1/health returns it as `version`.
ARG APP_VERSION=dev
ENV APP_VERSION=$APP_VERSION WEB_DIR=/app/web
USER nobody
EXPOSE 8080
ENTRYPOINT ["varlock", "run", "--", "/app/api"]
