-- Complete initial schema for the Kfz-Werkstatt customer portal.
-- Amounts are whole cents, timestamps are UTC.

CREATE TABLE IF NOT EXISTS customers (
    id         BIGSERIAL PRIMARY KEY,
    name       TEXT NOT NULL,
    email      TEXT NOT NULL UNIQUE,
    phone      TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS vehicles (
    id         BIGSERIAL PRIMARY KEY,
    plate      TEXT NOT NULL UNIQUE,
    brand      TEXT NOT NULL DEFAULT '',
    model      TEXT NOT NULL DEFAULT '',
    mileage    INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS orders (
    id             BIGSERIAL PRIMARY KEY,
    order_number   TEXT NOT NULL UNIQUE,
    customer_id    BIGINT NOT NULL REFERENCES customers (id),
    vehicle_id     BIGINT NOT NULL REFERENCES vehicles (id),
    status         TEXT NOT NULL DEFAULT 'requested',
    preferred_date DATE,
    description    TEXT NOT NULL DEFAULT '',
    labor_cents    BIGINT NOT NULL DEFAULT 0,
    parts_cents    BIGINT NOT NULL DEFAULT 0,
    net_cents      BIGINT NOT NULL DEFAULT 0,
    vat_cents      BIGINT NOT NULL DEFAULT 0,
    gross_cents    BIGINT NOT NULL DEFAULT 0,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS order_items (
    id               BIGSERIAL PRIMARY KEY,
    order_id         BIGINT NOT NULL REFERENCES orders (id) ON DELETE CASCADE,
    kind             TEXT NOT NULL,
    description      TEXT NOT NULL DEFAULT '',
    hours            NUMERIC(10, 2),
    quantity         NUMERIC(10, 2),
    unit_price_cents BIGINT NOT NULL DEFAULT 0,
    total_cents      BIGINT NOT NULL DEFAULT 0,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS order_status_history (
    id         BIGSERIAL PRIMARY KEY,
    order_id   BIGINT NOT NULL REFERENCES orders (id) ON DELETE CASCADE,
    status     TEXT NOT NULL,
    changed_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS employees (
    id            BIGSERIAL PRIMARY KEY,
    name          TEXT NOT NULL,
    email         TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS invoices (
    id          BIGSERIAL PRIMARY KEY,
    order_id    BIGINT NOT NULL UNIQUE REFERENCES orders (id),
    net_cents   BIGINT NOT NULL DEFAULT 0,
    vat_cents   BIGINT NOT NULL DEFAULT 0,
    gross_cents BIGINT NOT NULL DEFAULT 0,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS invoice_items (
    id               BIGSERIAL PRIMARY KEY,
    invoice_id       BIGINT NOT NULL REFERENCES invoices (id) ON DELETE CASCADE,
    kind             TEXT NOT NULL,
    description      TEXT NOT NULL DEFAULT '',
    hours            NUMERIC(10, 2),
    quantity         NUMERIC(10, 2),
    unit_price_cents BIGINT NOT NULL DEFAULT 0,
    total_cents      BIGINT NOT NULL DEFAULT 0,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS outbox (
    id              BIGSERIAL PRIMARY KEY,
    order_id        BIGINT REFERENCES orders (id),
    recipient       TEXT NOT NULL,
    subject         TEXT NOT NULL,
    body            TEXT NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    sent_at         TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_orders_status ON orders (status);
CREATE INDEX IF NOT EXISTS idx_orders_vehicle ON orders (vehicle_id);
CREATE INDEX IF NOT EXISTS idx_orders_customer ON orders (customer_id);
CREATE INDEX IF NOT EXISTS idx_order_status_history_order ON order_status_history (order_id);
CREATE INDEX IF NOT EXISTS idx_order_items_order ON order_items (order_id);
