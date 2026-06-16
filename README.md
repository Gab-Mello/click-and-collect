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
