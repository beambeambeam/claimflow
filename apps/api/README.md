# api

Go API built with [Gin](https://gin-gonic.com).

```sh
bun run dev    # go run . (listens on $PORT, default 8080)
bun run build  # go build -o dist/api .
bun run test   # go test ./...
```

## Endpoints

| Method | Path      | Response          |
| ------ | --------- | ----------------- |
| `GET`  | `/health` | `{"status":"ok"}` |
