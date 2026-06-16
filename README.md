# click-and-collect

A Go REST API for a college Click & Collect MVP.

The project implements a backend for MegaLoja Online, allowing customers to create orders using standard delivery or pickup in store.

## Quickstart

```bash
cp .env.example .env
make tidy   # first time only, pulls deps
make run
curl localhost:8080/healthz
```

## Common tasks

| Command             | Description               |
| ------------------- | ------------------------- |
| `make run`          | Run the API               |
| `make build`        | Build binary to `bin/api` |
| `make test`         | Run tests with `-race`    |
| `make fmt`          | Format code               |
| `make tidy`         | Tidy `go.mod` / `go.sum`  |
| `make migrate`      | Apply SQL migrations      |
| `make docs-lint`    | Lint `api/openapi.yaml`   |
| `make docs-preview` | Preview the spec with Redoc |

## Running with Docker

For the full stack (Postgres, migrations, API):

```bash
cp .env.example .env
docker compose up --build
```

Docker Compose starts Postgres first, then runs a one-shot `migrate` service that applies `migrations/0001_init.sql`, and finally starts the API.

This keeps the local development setup simple: after a fresh clone, or after running `docker compose down -v`, the app starts with the database schema and seed stores already loaded, without any manual database setup.

The migration is idempotent and intended for development only. In a production setup, migrations should be handled by a proper migration tool as part of the deployment process.

The API will be available at `http://localhost:8080`. Stop with `docker compose down`.

## API documentation

The API contract lives at [`api/openapi.yaml`](./api/openapi.yaml) (OpenAPI 3.1) and is the single source of truth for endpoints, schemas, and error codes.

When the server is running:

- Swagger UI: [`http://localhost:8080/docs`](http://localhost:8080/docs)
- Raw spec: [`http://localhost:8080/openapi.yaml`](http://localhost:8080/openapi.yaml)

## Frontend integration

The API base URL is:

```
http://localhost:8080/api/v1
```

CORS is enabled for these dev origins:

- `http://localhost:3000` (Next.js / CRA defaults)
- `http://localhost:5173` (Vite default)
