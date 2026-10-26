-- +goose Up
CREATE TABLE orders (
    id          uuid        PRIMARY KEY,
    customer_id uuid        NOT NULL REFERENCES users (id),
    status      text        NOT NULL CHECK (status IN ('open', 'cancelled')),
    note        text        NOT NULL DEFAULT '',
    total_cents bigint      NOT NULL DEFAULT 0 CHECK (total_cents >= 0),
    created_at  timestamptz NOT NULL DEFAULT now(),
    updated_at  timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX orders_customer_created_idx ON orders (customer_id, created_at DESC);
CREATE INDEX orders_status_idx ON orders (status);

CREATE TABLE order_items (
    id               uuid        PRIMARY KEY,
    order_id         uuid        NOT NULL REFERENCES orders (id) ON DELETE CASCADE,
    sku              text        NOT NULL,
    name             text        NOT NULL,
    quantity         integer     NOT NULL CHECK (quantity > 0),
    unit_price_cents bigint      NOT NULL CHECK (unit_price_cents >= 0),
    created_at       timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX order_items_order_id_idx ON order_items (order_id);

-- +goose Down
DROP TABLE order_items;
DROP TABLE orders;
