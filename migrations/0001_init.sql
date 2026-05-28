CREATE TABLE IF NOT EXISTS stores (
  id          TEXT PRIMARY KEY,
  name        TEXT NOT NULL,
  address     TEXT NOT NULL,
  city        TEXT NOT NULL,
  state       TEXT NOT NULL,
  zip         TEXT NOT NULL,
  capacity    INTEGER NOT NULL,
  active      BOOLEAN NOT NULL DEFAULT TRUE
);

CREATE TABLE IF NOT EXISTS orders (
  id              TEXT PRIMARY KEY,
  customer_name   TEXT NOT NULL,
  customer_email  TEXT NOT NULL,
  delivery_method TEXT NOT NULL,
  pickup_store_id TEXT REFERENCES stores(id),
  pickup_code     TEXT,
  status          TEXT NOT NULL,
  created_at      TIMESTAMPTZ NOT NULL,
  updated_at      TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS orders_pickup_store_id_idx ON orders(pickup_store_id);

CREATE TABLE IF NOT EXISTS notifications (
  id         TEXT PRIMARY KEY,
  order_id   TEXT NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
  message    TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS notifications_order_id_idx ON notifications(order_id);

INSERT INTO stores (id, name, address, city, state, zip, capacity, active) VALUES
  ('store-sp-paulista',   'MegaLoja Paulista',   'Av. Paulista, 1000', 'São Paulo',     'SP', '01310-100', 50, TRUE),
  ('store-rj-copacabana', 'MegaLoja Copacabana', 'Av. Atlântica, 500', 'Rio de Janeiro','RJ', '22010-000', 30, TRUE),
  ('store-mg-savassi',    'MegaLoja Savassi',    'R. Pernambuco, 200', 'Belo Horizonte','MG', '30130-150', 20, FALSE)
ON CONFLICT (id) DO NOTHING;
