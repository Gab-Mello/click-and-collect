# click-and-collect

A Go REST API for a college Click & Collect MVP.

The project implements a backend for MegaLoja Online, allowing customers to browse a product catalog, build a persistent cart, and convert it into an order with either standard delivery or pickup in store.

```text
Products → Cart → Checkout → Order → Pickup code / status / notification
```

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

Docker Compose starts Postgres first, then runs a one-shot `migrate` service that applies every `migrations/*.sql` file in order (currently `0001_init.sql` for stores/orders/notifications and `0002_products_carts_checkout.sql` for products/carts/order_items plus the seeded product catalog), and finally starts the API.

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

### Product images

This is an academic MVP, so product image handling is deliberately kept as
simple as possible: each product carries an `image_url` field that is just a
path string (e.g. `/products/headphones.jpg`). The backend only stores and
returns that path — it does **not** upload, process, host, or serve the
image bytes themselves.

The frontend is responsible for providing the static files under its own
public directory:

```text
frontend/public/products/smartphone.jpg
frontend/public/products/notebook.jpg
frontend/public/products/headphones.jpg
frontend/public/products/mouse.jpg
frontend/public/products/keyboard.jpg
```

Then in the UI:

```jsx
<img src={product.image_url} alt={product.name} />
```

This split sidesteps a lot of complexity that isn't relevant to the goal of
the project: file uploads, object storage, CDN integration, image
optimization, and backend static file hosting. A real production
e-commerce system would typically push images to an object store / CDN and
likely model product media as its own resource — but for a college MVP
whose goal is to demonstrate the Click & Collect flow (not a full product
media pipeline), frontend-controlled static paths are enough.

## End-to-end flow (curl)

```bash
# 1. browse the catalog
curl -s localhost:8080/api/v1/products | jq .

# 2. create a cart
CART=$(curl -s -X POST localhost:8080/api/v1/carts \
  -H 'content-type: application/json' \
  -d '{"customer_email":"grace@example.com"}' | jq -r .id)

# 3. add items (re-posting the same product upserts the quantity)
curl -s -X POST localhost:8080/api/v1/carts/$CART/items \
  -H 'content-type: application/json' \
  -d '{"product_id":"prod-headphones","quantity":2}'
curl -s -X POST localhost:8080/api/v1/carts/$CART/items \
  -H 'content-type: application/json' \
  -d '{"product_id":"prod-mouse","quantity":1}'

# 4. checkout (pickup in store)
curl -s -X POST localhost:8080/api/v1/carts/$CART/checkout \
  -H 'content-type: application/json' \
  -d '{
        "customer_name":"Grace Hopper",
        "customer_email":"grace@example.com",
        "delivery_method":"pickup_in_store",
        "pickup_store_id":"store-sp-paulista"
      }' | jq '.order | {id, total_amount_cents, pickup_code, items}'

# 5. once preparation is done, transition the order — this also emits
#    a pickup-ready notification for pickup_in_store orders
ORDER=...  # from the previous response
curl -s -X PATCH localhost:8080/api/v1/orders/$ORDER/status \
  -H 'content-type: application/json' \
  -d '{"status":"READY_FOR_PICKUP"}' | jq .
```
