# click-and-collect

A minimal Go REST API starter for learning and experimentation.

## Quickstart

```bash
cp .env.example .env
make tidy   # first time only — pulls deps
make run
curl localhost:8080/healthz
```

## Layout

```
cmd/api/                # entry point
internal/config/        # env-based config loading
internal/server/        # http.Server + chi router + graceful shutdown
```

Domain packages (e.g. `internal/orders/`) are added as features arrive — each holds its own `handler.go`, `service.go`, `store.go`, `model.go`.

## Common tasks

| Command       | Description                |
| ------------- | -------------------------- |
| `make run`    | Run the API                |
| `make build`  | Build binary to `bin/api`  |
| `make test`   | Run tests with `-race`     |
| `make fmt`    | Format code                |
| `make tidy`   | Tidy `go.mod` / `go.sum`   |

## Conventions

- Stdlib first (`net/http`, `log/slog`, `context`); `chi` only for routing/middleware.
- Concrete types until a seam is needed — no interfaces just for the sake of it.
- Config is loaded once, in `main`, and passed down explicitly.
- Graceful shutdown on `SIGINT` / `SIGTERM`.
