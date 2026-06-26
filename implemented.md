# Implemented

An inventory of what has actually been built in this repository. For setup and run instructions see [README.md](./README.md); for the HTTP contract see [api/openapi.yaml](./api/openapi.yaml). This document is a reference for what exists in the code today.

## Overview

A Go 1.26 REST API backend for the **MegaLoja Online Click & Collect** MVP. It exposes endpoints for browsing a product catalog and physical pickup stores, building a persistent shopping cart, converting that cart into an order via a transactional checkout, and walking the order through its lifecycle (with pickup-ready notifications for in-store orders).

The end-to-end customer flow:

```
Products → Cart → Checkout → Order (items + total) → Pickup code / status / notification
```

## Tech Stack

- **Language**: Go 1.26
- **HTTP router**: `github.com/go-chi/chi/v5` v5.1.0
- **Database driver**: `github.com/jackc/pgx/v5` v5.9.2 (pool-based)
- **Database**: PostgreSQL 16
- **Logging**: `log/slog` (standard library, JSON handler)
- **API spec**: OpenAPI 3.1.0 (hand-maintained YAML)
- **Swagger UI**: v5.17.14 (pinned, served from jsDelivr)
- **Container runtime**: `gcr.io/distroless/static-debian12:nonroot`

## Project Structure

```
click-and-collect/
├── api/
│   ├── openapi.yaml         # OpenAPI 3.1 contract
│   └── embed.go             # //go:embed of the spec
├── cmd/api/
│   └── main.go              # Application entry point
├── internal/
│   ├── config/              # Env-driven config
│   ├── db/                  # pgx pool bootstrap
│   ├── docs/                # /docs and /openapi.yaml handlers
│   ├── httpx/               # JSON response & error helpers
│   ├── server/              # Server lifecycle, routing, CORS
│   ├── stores/              # Stores domain (read-only)
│   ├── products/            # Products domain (read-only)
│   ├── carts/               # Carts domain (active cart + transactional checkout)
│   └── orders/              # Orders domain (full lifecycle, incl. CheckoutTx)
├── migrations/
│   ├── 0001_init.sql                       # Stores / orders / notifications schema + seed
│   └── 0002_products_carts_checkout.sql    # Products / carts / cart_items / order_items + product seed
├── Dockerfile               # Multi-stage build → distroless
├── docker-compose.yml       # Postgres + one-shot migrate + api
├── Makefile                 # Dev tasks
├── redocly.yaml             # OpenAPI lint config
├── .env / .env.example      # Local configuration
└── README.md
```

## Architecture

Clean layered architecture, top to bottom:

```
HTTP Handler  →  Service (business logic + validation)  →  Repository (SQL)  →  PostgreSQL
```

- Domain models (`model.go` in each package) are separate from JSON request/response DTOs (defined in the handler files).
- Typed sentinel errors are produced in the service layer and translated to HTTP status codes by the handler layer.
- Packages under `internal/` are not importable outside the module.

## Domains Implemented

### Stores (`internal/stores/`)

Read-only catalog of physical pickup locations. Stores are not created via the API — they are seeded by the initial migration.

- `Store` model: `ID`, `Name`, `Address`, `City`, `State`, `ZIP`, `Capacity`, `Active`.
- Service operations: `List`, `Get` (returns `ErrNotFound` if missing).
- Inactive stores are returned by the list endpoint but rejected by the orders service when used as a pickup target.

### Products (`internal/products/`)

Read-only catalog of purchasable items. Products are seeded by `migrations/0002_products_carts_checkout.sql`; there is no admin CRUD.

- `Product` model: `ID`, `Name`, `Description`, `PriceCents` (int64, BRL cents), `ImageURL`, `Active`, `CreatedAt`, `UpdatedAt`.
- Service operations: `List`, `Get` (returns `ErrNotFound`).
- **Image handling is intentionally out of scope**: `ImageURL` is just a path string (e.g. `/products/headphones.jpg`). The backend does not upload, host, or serve image binaries — the frontend is expected to place the files under its own static directory.
- Seeded products: `prod-smartphone`, `prod-notebook`, `prod-headphones`, `prod-mouse`, `prod-keyboard`.

### Carts (`internal/carts/`)

Persistent shopping cart with a transactional checkout.

- `Cart` model: `ID` (16-char hex), `CustomerEmail`, `Status` (`ACTIVE` | `CHECKED_OUT`), `Items`, `CreatedAt`, `UpdatedAt`.
- `CartItem` model: `ID`, `CartID`, `ProductID`, `Quantity`, `CreatedAt`, `UpdatedAt`. A `UNIQUE (cart_id, product_id)` constraint at the DB level guarantees one row per product per cart.
- **Lifecycle**: a cart starts `ACTIVE`. While active, items can be added (upsert: re-adding the same product **increases** quantity), patched to a new quantity, or removed. Once checked out, the cart becomes `CHECKED_OUT` and any modification attempt returns `409 cart_not_active`.
- **Checkout** (`POST /api/v1/carts/{id}/checkout`) wraps the entire conversion in a single `pgx.Tx` (`pgx.BeginFunc` from `internal/carts/service.go`):
  1. `SELECT … FOR UPDATE` on the cart row prevents racing double-checkouts.
  2. Validate cart is `ACTIVE` and has ≥1 item.
  3. Look up every product; reject if any is inactive.
  4. Snapshot product `name` + `unit_price_cents` into `order_items` — past orders are immune to future product changes.
  5. Server computes `total_amount_cents` from the snapshots.
  6. Call `orders.Service.CheckoutTx`, which runs the same customer/delivery/pickup-store validation as the legacy `POST /api/v1/orders`.
  7. Mark the cart `CHECKED_OUT`.

  Any error rolls the whole transaction back; the cart stays `ACTIVE` and no partial order is persisted.

### Orders (`internal/orders/`)

Core domain. Covers creation, retrieval, status transitions, and notifications.

- `Order` model: `ID` (16-char hex), `CustomerName`, `CustomerEmail`, `DeliveryMethod`, `PickupStoreID?`, `PickupCode?`, `Status`, `CartID?`, `TotalAmountCents`, `Items`, `CreatedAt`, `UpdatedAt`.
- `OrderItem` model: `ID`, `OrderID`, `ProductID`, `ProductName` (snapshot), `UnitPriceCents` (snapshot), `Quantity`, `TotalPriceCents`.
- `Notification` model: `ID`, `OrderID`, `Message`, `CreatedAt`.
- **Delivery methods**: `standard` and `pickup_in_store`. Pickup orders must reference an active store; standard orders must not reference one.
- **Pickup code**: 6-digit zero-padded random number, generated on creation for `pickup_in_store` orders only.
- **Order ID**: cryptographically random 8 bytes → 16-char hex string.
- **Status state machine** (enforced by `canTransition` in `internal/orders/service.go`):

  ```
  AWAITING_PREPARATION ──▶ READY_FOR_PICKUP ──▶ COLLECTED
            │                       │
            └────────▶ CANCELLED ◀──┘
  ```

- **Automatic notification**: when a `pickup_in_store` order transitions to `READY_FOR_PICKUP`, the service creates a notification with the message `Order {ID} is ready for pickup. Use code {code}.`
- **Two creation paths**:
  - `POST /api/v1/orders` — **legacy**, creates an order **without items** (`items: []`, `total_amount_cents: 0`). Kept for backward compatibility.
  - `POST /api/v1/carts/{id}/checkout` — **canonical**, creates an order with items and a server-computed total, driven by the carts service.

## HTTP API

All non-health/non-docs routes are mounted under `/api/v1`. Routing lives in `internal/server/router.go`.

| Method | Path                                              | Purpose                                                |
|--------|---------------------------------------------------|--------------------------------------------------------|
| GET    | `/healthz`                                        | Unversioned liveness probe                             |
| GET    | `/api/v1/health`                                  | Versioned liveness probe                               |
| GET    | `/api/v1/stores`                                  | List all stores (including inactive)                   |
| GET    | `/api/v1/stores/{id}`                             | Get a single store by ID                               |
| GET    | `/api/v1/products`                                | List all products (including inactive)                 |
| GET    | `/api/v1/products/{id}`                           | Get a single product by ID                             |
| POST   | `/api/v1/carts`                                   | Create a new active cart                               |
| GET    | `/api/v1/carts/{id}`                              | Get a cart (items hydrated)                            |
| POST   | `/api/v1/carts/{id}/items`                        | Add item to cart (upserts quantity)                    |
| PATCH  | `/api/v1/carts/{id}/items/{product_id}`           | Set absolute quantity of a cart line                   |
| DELETE | `/api/v1/carts/{id}/items/{product_id}`           | Remove a line from the cart                            |
| POST   | `/api/v1/carts/{id}/checkout`                     | Transactional checkout → creates an order with items   |
| POST   | `/api/v1/orders`                                  | **Legacy**: create an empty-item order directly        |
| GET    | `/api/v1/orders/{id}`                             | Get an order by ID (items hydrated)                    |
| PATCH  | `/api/v1/orders/{id}/status`                      | Transition an order's status                           |
| GET    | `/api/v1/orders/{id}/notifications`               | List notifications for an order                        |
| GET    | `/openapi.yaml`                                   | Raw OpenAPI 3.1 spec                                   |
| GET    | `/docs`                                           | Swagger UI                                             |

Health responses are `{"status":"ok"}`. Status updates return both the updated order and the (optional) notification that was created as a side-effect.

## Validation & Error Handling

Service-layer validation surfaces typed errors. Each domain's handler maps them to snake_case `code` strings and HTTP status codes:

| Sentinel (package) | HTTP | `code` |
|---|---|---|
| `orders.ErrNotFound` | 404 | `order_not_found` |
| `stores.ErrNotFound` | 404 | `store_not_found` |
| `products.ErrNotFound` | 404 | `product_not_found` |
| `carts.ErrCartNotFound` | 404 | `cart_not_found` |
| `carts.ErrCartItemNotFound` | 404 | `cart_item_not_found` |
| `orders.ErrInvalidDeliveryMethod` | 400 | `invalid_delivery_method` |
| `orders.ErrCustomerNameRequired` | 400 | `customer_name_required` |
| `orders.ErrCustomerEmailRequired` / `carts.ErrCustomerEmailReq` | 400 | `customer_email_required` |
| `orders.ErrPickupStoreRequired` | 400 | `pickup_store_required` |
| `orders.ErrPickupStoreNotAllowed` | 400 | `pickup_store_not_allowed` |
| `orders.ErrInvalidStatus` | 400 | `invalid_status` |
| `carts.ErrInvalidQuantity` | 400 | `invalid_quantity` |
| `orders.ErrStoreInactive` | 409 | `store_inactive` |
| `orders.ErrInvalidTransition` | 409 | `invalid_transition` |
| `carts.ErrCartNotActive` | 409 | `cart_not_active` |
| `carts.ErrCartEmpty` | 409 | `cart_empty` |
| `carts.ErrProductInactive` | 409 | `product_inactive` |

Handlers translate these into a standardized JSON error envelope produced by `internal/httpx/respond.go`:

```json
{ "error": "human-readable message", "code": "machine_code" }
```

HTTP status mapping used by the handlers:

- `400 Bad Request` — input/shape validation failures
- `404 Not Found` — unknown order or store
- `409 Conflict` — business-rule violations (inactive store, invalid transition)
- `500 Internal Server Error` — unexpected failures

## Persistence Layer

PostgreSQL accessed via a `pgxpool.Pool` configured in `internal/db/db.go` (max 8 connections, 1-hour max conn lifetime, 5s connect timeout).

Schema and seed data are defined idempotently across two migration files:

**`migrations/0001_init.sql`**

- **`stores`** — `id` (PK), `name`, `address`, `city`, `state`, `zip`, `capacity`, `active` (default `TRUE`).
- **`orders`** — `id` (PK), `customer_name`, `customer_email`, `delivery_method`, `pickup_store_id` (FK → `stores.id`), `pickup_code`, `status`, `created_at`, `updated_at`. Index: `orders_pickup_store_id_idx`.
- **`notifications`** — `id` (PK), `order_id` (FK → `orders.id` `ON DELETE CASCADE`), `message`, `created_at`. Index: `notifications_order_id_idx`.

**`migrations/0002_products_carts_checkout.sql`**

- **`products`** — `id` (PK), `name`, `description`, `price_cents` (`BIGINT NOT NULL CHECK >= 0`), `image_url`, `active`, `created_at`, `updated_at` (default `now()`).
- **`carts`** — `id` (PK), `customer_email`, `status`, `created_at`, `updated_at`.
- **`cart_items`** — `id` (PK), `cart_id` (FK → `carts.id` `ON DELETE CASCADE`), `product_id` (FK → `products.id`), `quantity` (`CHECK > 0`), `created_at`, `updated_at`. Indexes on both FKs; `UNIQUE (cart_id, product_id)` enforces one row per product per cart and powers the upsert behavior.
- **`orders`** gains `total_amount_cents` (`BIGINT NOT NULL DEFAULT 0`) and `cart_id` (FK → `carts.id`, nullable) via `ALTER TABLE … ADD COLUMN IF NOT EXISTS`. Index: `orders_cart_id_idx`.
- **`order_items`** — `id` (PK), `order_id` (FK → `orders.id` `ON DELETE CASCADE`), `product_id` (FK → `products.id`), `product_name` (snapshot), `unit_price_cents` (snapshot), `quantity` (`CHECK > 0`), `total_price_cents`. Indexes on both FKs.

Seeded stores (`0001`):

- `store-sp-paulista` — MegaLoja Paulista, São Paulo, capacity 50, active
- `store-rj-copacabana` — MegaLoja Copacabana, Rio de Janeiro, capacity 30, active
- `store-mg-savassi` — MegaLoja Savassi, Belo Horizonte, capacity 20, **inactive**

Seeded products (`0002`): `prod-smartphone`, `prod-notebook`, `prod-headphones`, `prod-mouse`, `prod-keyboard`. Image URLs point at `/products/<slug>.jpg`, expected to be served by the frontend.

Both migration files use `CREATE TABLE IF NOT EXISTS`, `ALTER TABLE … ADD COLUMN IF NOT EXISTS`, and `INSERT … ON CONFLICT DO NOTHING`, so re-running them is a no-op.

### Transactional checkout

The single point in the system that requires a transaction is `carts.Service.Checkout`. It uses `pgx.BeginFunc(ctx, pool, func(tx pgx.Tx) error { ... })` and passes the resulting `pgx.Tx` into both `carts.Repo` `*Tx` methods (`LockCartTx`, `ListItemsTx`, `MarkCheckedOutTx`) and `orders.Service.CheckoutTx`. Both `*pgxpool.Pool` and `pgx.Tx` satisfy a small local `dbtx` querier interface declared in each repo, so the same SQL helpers run in both contexts without a UoW abstraction.

## API Documentation

- The OpenAPI 3.1 spec lives at `api/openapi.yaml` and is embedded into the binary via `//go:embed` in `api/embed.go`, so the running container does not need the file on disk.
- `internal/docs/docs.go` serves the embedded spec at `GET /openapi.yaml` and a Swagger UI page at `GET /docs` (Swagger UI v5.17.14, pinned).
- `redocly.yaml` configures linting using the `recommended` preset with a small set of MVP-appropriate rule exceptions (`security-defined`, `info-license`, `no-server-example.com`, `operation-4xx-response` disabled).
- `make docs-lint` and `make docs-preview` are provided for working on the spec.

## Server, Middleware & Configuration

**Server lifecycle** (`internal/server/server.go`): builds an `http.Server` with `ReadTimeout` 5s, `WriteTimeout` 10s, `IdleTimeout` 60s, and shuts down gracefully on `SIGINT`/`SIGTERM` with a 10s shutdown timeout.

**Middleware chain** (`internal/server/router.go`):

- `chi/middleware.RequestID`
- `chi/middleware.RealIP`
- `chi/middleware.Logger`
- `chi/middleware.Recoverer`
- Custom CORS (`internal/server/cors.go`) — allows `http://localhost:3000` and `http://localhost:5173`; methods `GET, POST, PATCH, DELETE, OPTIONS`; headers `Content-Type, Authorization`; `Max-Age: 300`.

**Configuration** (`internal/config/config.go`) is read from environment variables with sensible defaults:

| Variable        | Default                                                                          | Notes        |
|-----------------|----------------------------------------------------------------------------------|--------------|
| `APP_ENV`       | `local`                                                                          |              |
| `PORT`          | `8080`                                                                           |              |
| `LOG_LEVEL`     | `info`                                                                           |              |
| `DATABASE_URL`  | `postgres://postgres:postgres@localhost:5432/click_collect?sslmode=disable`      | Required     |

**Logging** is JSON-structured via `log/slog`, with level driven by `LOG_LEVEL`.

## Build & Deployment

**Dockerfile** — multi-stage build:

1. `golang:1.26-alpine` builds a static binary (`CGO_ENABLED=0`, `-trimpath`, stripped symbols), caching modules separately.
2. `gcr.io/distroless/static-debian12:nonroot` runs the binary as the unprivileged `nonroot` user. Port `8080` exposed.

**docker-compose.yml** — three services with healthcheck-gated startup:

- `postgres` (postgres:16-alpine) — persistent `pgdata` volume, `pg_isready` healthcheck.
- `migrate` — one-shot container that loops over every `migrations/*.sql` file in order (currently `0001_init.sql` and `0002_products_carts_checkout.sql`). Depends on `postgres` being healthy.
- `api` — built from the local `Dockerfile`, depends on `postgres` (healthy) and `migrate` (completed successfully), reads `.env` with `DATABASE_URL` overridden to point at the in-network `postgres` host.

**Makefile** targets: `run`, `build` (to `bin/api`), `test` (`go test ./... -race -count=1`), `fmt`, `tidy`, `migrate`, `docs-lint`, `docs-preview`.

## Not Yet Implemented / Known Gaps

- **No automated tests.** The `make test` target exists but there are currently no `*_test.go` files in the repo.
- **No authentication or authorization.** The API is open; no security schemes are declared in the OpenAPI spec.
- **No rate limiting, metrics, or tracing.** Only request logging is wired in.
- **No payment, no stock control, no shipping calculation, no coupons.** The MVP stops at "order created" — fulfilment is simulated by status transitions and a generated notification.
- **No real email/SMS delivery.** The `READY_FOR_PICKUP` notification is persisted in the database; no external channel is contacted.
- **No image hosting.** `image_url` is a path string only — the backend never reads, writes, or serves the image bytes.
- **No admin CRUD for products or stores.** Both are read-only and seeded by migrations.
- **Notification idempotency** is not enforced — replaying a status update to `READY_FOR_PICKUP` could in principle create duplicate notifications. Acceptable for the MVP scope.
- **Pickup code uniqueness** is not enforced at the database level; collision probability is acceptable for the MVP but not guaranteed.
- **Migration tooling**: two idempotent SQL scripts run by `psql` in a Docker one-shot. No formal versioning, rollback, or external migration tool.
