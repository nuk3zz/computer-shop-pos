BEGIN;

ALTER TABLE orders ADD COLUMN IF NOT EXISTS fulfillment_type VARCHAR(30) NOT NULL DEFAULT 'in_store';
ALTER TABLE orders ADD COLUMN IF NOT EXISTS fulfillment_status VARCHAR(30) NOT NULL DEFAULT 'completed';
ALTER TABLE orders ADD COLUMN IF NOT EXISTS stock_committed BOOLEAN NOT NULL DEFAULT false;

CREATE INDEX IF NOT EXISTS idx_orders_fulfillment ON orders(order_type, fulfillment_status);

COMMIT;
