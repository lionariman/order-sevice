-- Заказы: только «шапка»
CREATE TABLE IF NOT EXISTS orders (
  order_uid          TEXT PRIMARY KEY,
  track_number       TEXT NOT NULL,
  entry              TEXT NOT NULL,
  locale             TEXT NOT NULL,
  internal_signature TEXT NOT NULL,
  customer_id        TEXT NOT NULL,
  delivery_service   TEXT NOT NULL,
  shardkey           TEXT NOT NULL,
  sm_id              INTEGER NOT NULL,
  date_created       TIMESTAMPTZ NOT NULL,
  oof_shard          TEXT NOT NULL,
  updated_at         TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Доставка: один к одному к заказу
CREATE TABLE IF NOT EXISTS deliveries (
  order_uid TEXT PRIMARY KEY REFERENCES orders(order_uid) ON DELETE CASCADE,
  name      TEXT NOT NULL,
  phone     TEXT NOT NULL,
  zip       TEXT NOT NULL,
  city      TEXT NOT NULL,
  address   TEXT NOT NULL,
  region    TEXT NOT NULL,
  email     TEXT NOT NULL
);

-- Оплата: один к одному к заказу
CREATE TABLE IF NOT EXISTS payments (
  order_uid    TEXT PRIMARY KEY REFERENCES orders(order_uid) ON DELETE CASCADE,
  transaction  TEXT NOT NULL UNIQUE,
  request_id   TEXT NOT NULL,
  currency     TEXT NOT NULL,
  provider     TEXT NOT NULL,
  amount       INTEGER NOT NULL,
  payment_dt   BIGINT  NOT NULL,
  bank         TEXT NOT NULL,
  delivery_cost INTEGER NOT NULL,
  goods_total   INTEGER NOT NULL,
  custom_fee    INTEGER NOT NULL
);

-- Товары: один ко многим к заказу
CREATE TABLE IF NOT EXISTS items (
  order_uid    TEXT    NOT NULL REFERENCES orders(order_uid) ON DELETE CASCADE,
  chrt_id      INTEGER NOT NULL,
  track_number TEXT    NOT NULL,
  price        INTEGER NOT NULL,
  rid          TEXT    NOT NULL,
  name         TEXT    NOT NULL,
  sale         INTEGER NOT NULL,
  size         TEXT    NOT NULL,
  total_price  INTEGER NOT NULL,
  nm_id        INTEGER NOT NULL,
  brand        TEXT    NOT NULL,
  status       INTEGER NOT NULL,
  PRIMARY KEY (order_uid, chrt_id)
);

CREATE INDEX IF NOT EXISTS idx_orders_updated_at ON orders(updated_at DESC);
CREATE INDEX IF NOT EXISTS idx_items_order ON items(order_uid);
