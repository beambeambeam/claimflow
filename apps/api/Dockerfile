# syntax=docker/dockerfile:1

FROM ghcr.io/dmno-dev/varlock:1.24.0 AS varlock

FROM golang:1.27.3 AS build
WORKDIR /src

COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download

COPY *.go ./
COPY env ./env
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/api .

# The varlock binary needs libstdc++ and CA certificates, so the image uses Alpine instead of distroless.
FROM alpine:3.24
RUN apk add --no-cache ca-certificates libstdc++
COPY --from=varlock /usr/local/bin/varlock /usr/local/bin/varlock
WORKDIR /app
COPY .env.schema ./
COPY --from=build /out/api ./api
USER nobody
EXPOSE 8080
ENTRYPOINT ["varlock", "run", "--", "/app/api"]
