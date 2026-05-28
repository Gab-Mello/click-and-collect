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

Domain packages (e.g. `internal/orders/`) are added as features arrive — each holds its own `handler.go`, `service.go`, `repository.go`, `model.go`.

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

## Running with Docker

For local development with the full stack (API + Postgres):

```bash
cp .env.example .env
docker compose up --build
```

The API will be available at:

```
http://localhost:8080
```

Test it:

```bash
curl http://localhost:8080/healthz
curl http://localhost:8080/api/v1/stores
```

To stop the stack:

```bash
docker compose down
```

To reset the database volume too:

```bash
docker compose down -v
```

### Database mode

The API currently runs with **in-memory repositories** seeded with sample stores — Postgres is launched by compose but not yet wired to the API. The `DATABASE_URL` config is in place; PostgreSQL persistence will be added in a follow-up step.

### Host vs container hostnames

`DATABASE_URL` uses different hosts depending on where the API runs:

- API on your host, Postgres in Docker: `postgres://postgres:postgres@localhost:5432/click_collect?sslmode=disable` (this is what `.env.example` has).
- Both API and Postgres in Docker: `compose` overrides the host to `postgres` (the service name on the compose network).

## Frontend integration

The API base URL is:

```
http://localhost:8080/api/v1
```

CORS is enabled for the following dev origins:

- `http://localhost:3000` (Next.js / CRA defaults)
- `http://localhost:5173` (Vite default)

If you run the frontend on a different port, update `internal/server/cors.go`.
