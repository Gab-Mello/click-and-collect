CREATE TABLE IF NOT EXISTS products (
  id          TEXT PRIMARY KEY,
  name        TEXT NOT NULL,
  description TEXT NOT NULL DEFAULT '',
  price_cents BIGINT NOT NULL CHECK (price_cents >= 0),
  image_url   TEXT NOT NULL DEFAULT '',
  active      BOOLEAN NOT NULL DEFAULT TRUE,
  created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS carts (
  id             TEXT PRIMARY KEY,
  customer_email TEXT NOT NULL,
  status         TEXT NOT NULL,
  created_at     TIMESTAMPTZ NOT NULL,
  updated_at     TIMESTAMPTZ NOT NULL
);

CREATE TABLE IF NOT EXISTS cart_items (
  id         TEXT PRIMARY KEY,
  cart_id    TEXT NOT NULL REFERENCES carts(id) ON DELETE CASCADE,
  product_id TEXT NOT NULL REFERENCES products(id),
  quantity   INTEGER NOT NULL CHECK (quantity > 0),
  created_at TIMESTAMPTZ NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL,
  UNIQUE (cart_id, product_id)
);

CREATE INDEX IF NOT EXISTS cart_items_cart_id_idx    ON cart_items(cart_id);
CREATE INDEX IF NOT EXISTS cart_items_product_id_idx ON cart_items(product_id);

ALTER TABLE orders ADD COLUMN IF NOT EXISTS total_amount_cents BIGINT NOT NULL DEFAULT 0;
ALTER TABLE orders ADD COLUMN IF NOT EXISTS cart_id            TEXT REFERENCES carts(id);

CREATE INDEX IF NOT EXISTS orders_cart_id_idx ON orders(cart_id);

CREATE TABLE IF NOT EXISTS order_items (
  id                TEXT PRIMARY KEY,
  order_id          TEXT NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
  product_id        TEXT NOT NULL REFERENCES products(id),
  product_name      TEXT NOT NULL,
  unit_price_cents  BIGINT NOT NULL CHECK (unit_price_cents >= 0),
  quantity          INTEGER NOT NULL CHECK (quantity > 0),
  total_price_cents BIGINT NOT NULL CHECK (total_price_cents >= 0)
);

CREATE INDEX IF NOT EXISTS order_items_order_id_idx   ON order_items(order_id);
CREATE INDEX IF NOT EXISTS order_items_product_id_idx ON order_items(product_id);

INSERT INTO products (id, name, description, price_cents, image_url, active, created_at, updated_at) VALUES
  ('prod-smartphone',    'Smartphone Galaxy A55',  '6.6" 256GB display, 8GB RAM',             199900, '/products/smartphone.jpg',    TRUE, now(), now()),
  ('prod-notebook',      'Notebook Pro 14',        'Ryzen 7, 16GB RAM, 512GB SSD',            549900, '/products/notebook.jpg',      TRUE, now(), now()),
  ('prod-headphones',    'Wireless Headphones X1', 'Bluetooth 5.3 with active noise cancel',   59900, '/products/headphones.jpg',    TRUE, now(), now()),
  ('prod-mouse',         'Wireless Mouse M2',      'Ergonomic 2.4GHz, 18-month battery',        9900, '/products/mouse.jpg',         TRUE, now(), now()),
  ('prod-keyboard',      'Mechanical Keyboard K1', 'Hot-swap switches, RGB backlighting',      29900, '/products/keyboard.jpg',      TRUE, now(), now()),
  ('prod-monitor',       'Monitor UltraView 27',   '27" QHD 165Hz IPS, HDR400',               179900, '/products/monitor.jpg',       TRUE, now(), now()),
  ('prod-tablet',        'Tablet Slim 11',         '11" 128GB Wi-Fi, octa-core',              229900, '/products/tablet.jpg',        TRUE, now(), now()),
  ('prod-smartwatch',    'Smartwatch Pulse S',     'GPS, heart rate, 5ATM water resistant',    79900, '/products/smartwatch.jpg',    TRUE, now(), now()),
  ('prod-webcam',        'Webcam Studio 4K',       '4K UHD, dual-mic, auto light correction',  39900, '/products/webcam.jpg',        TRUE, now(), now()),
  ('prod-speaker',       'Bluetooth Speaker Boom', 'IPX7 portable, 24h playtime',              24900, '/products/speaker.jpg',       TRUE, now(), now()),
  ('prod-external-ssd',  'External SSD 1TB',       'USB-C 3.2 Gen 2, 1050 MB/s',               69900, '/products/external-ssd.jpg',  TRUE, now(), now()),
  ('prod-usb-c-charger', 'USB-C Charger 65W',      'GaN, dual port, fast charge',              19900, '/products/usb-c-charger.jpg', TRUE, now(), now()),
  ('prod-gaming-chair',  'Gaming Chair Apex',      'Reclining 180°, lumbar support',          129900, '/products/gaming-chair.jpg',  TRUE, now(), now()),
  ('prod-printer',       'Printer EcoTank M200',   'Wi-Fi multifunction, refill tank',         89900, '/products/printer.jpg',       TRUE, now(), now()),
  ('prod-wifi-router',   'Wi-Fi 6 Router AX3000',  'Dual-band, MU-MIMO, 4 LAN ports',          49900, '/products/wifi-router.jpg',   TRUE, now(), now())
ON CONFLICT (id) DO NOTHING;
