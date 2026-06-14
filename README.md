# click-and-collect

A Go REST API for a college Click & Collect MVP.

The project implements a backend for MegaLoja Online, allowing customers to create orders using standard delivery or pickup in store.


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

| Command         | Description                |
| --------------- | -------------------------- |
| `make run`      | Run the API                |
| `make build`    | Build binary to `bin/api`  |
| `make test`     | Run tests with `-race`     |
| `make fmt`      | Format code                |
| `make tidy`     | Tidy `go.mod` / `go.sum`   |
| `make migrate`  | Apply SQL migrations       |
| `make docs-lint`    | Lint `api/openapi.yaml`         |
| `make docs-preview` | Preview the spec with Redoc     |

## Conventions

- Stdlib first (`net/http`, `log/slog`, `context`); `chi` only for routing/middleware.
- Concrete types until a seam is needed — no interfaces just for the sake of it.
- Config is loaded once, in `main`, and passed down explicitly.
- Graceful shutdown on `SIGINT` / `SIGTERM`.

## Running with Docker

For local development with the full stack (Postgres + migrations + API):

```bash
cp .env.example .env
docker compose up --build
```

Compose runs three services in order: `postgres` (waits for healthcheck) → `migrate` (one-shot, applies `migrations/0001_init.sql`) → `api`. The API only starts after migrations have completed successfully.

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

### Database

All data is persisted in Postgres — `stores`, `orders`, and `notifications`. The API requires `DATABASE_URL` to be set and will refuse to start without it.

Migrations live in `migrations/0001_init.sql` and include the seed stores. They are idempotent (safe to re-run): `CREATE TABLE IF NOT EXISTS` and `INSERT ... ON CONFLICT DO NOTHING`.

To apply migrations against a host or remote Postgres (e.g. when running the API directly with `make run`), use:

```bash
make migrate   # requires psql + DATABASE_URL set in your env
```

`docker compose down -v` wipes the database volume — the next `up` will re-seed via the `migrate` service.

### Host vs container hostnames

`DATABASE_URL` uses different hosts depending on where the API runs:

- API on your host, Postgres in Docker: `postgres://postgres:postgres@localhost:5432/click_collect?sslmode=disable` (this is what `.env.example` has).
- Both API and Postgres in Docker: `compose` overrides the host to `postgres` (the service name on the compose network).

## API documentation

The API contract lives at [`api/openapi.yaml`](./api/openapi.yaml) (OpenAPI 3.1) and is the single source of truth for endpoints, schemas, and error codes.

When the server is running, browse to:

- **Swagger UI** — [`http://localhost:8080/docs`](http://localhost:8080/docs)
- **Raw spec** — [`http://localhost:8080/openapi.yaml`](http://localhost:8080/openapi.yaml)

The spec is hand-maintained — when you change a handler, update `api/openapi.yaml` in the same PR. Run `make docs-lint` before committing.

## Frontend integration

The API base URL is:

```
http://localhost:8080/api/v1
```

CORS is enabled for the following dev origins:

- `http://localhost:3000` (Next.js / CRA defaults)
- `http://localhost:5173` (Vite default)

If you run the frontend on a different port, update `internal/server/cors.go`.
