-- Product cost/profit, explicit item type, and repair-customer contact details.
BEGIN;

ALTER TABLE products ADD COLUMN IF NOT EXISTS cost_price DECIMAL(10,2) NOT NULL DEFAULT 0;
ALTER TABLE products ADD COLUMN IF NOT EXISTS item_type VARCHAR(20) NOT NULL DEFAULT 'product';
ALTER TABLE products DROP CONSTRAINT IF EXISTS products_item_type_check;
ALTER TABLE products ADD CONSTRAINT products_item_type_check CHECK (item_type IN ('product', 'service'));

UPDATE products p
SET cost_price = COALESCE(i.unit_cost, 0),
    item_type = CASE WHEN p.preparation_time > 0 THEN 'service' ELSE 'product' END
FROM inventory i
WHERE i.product_id = p.id;

UPDATE products
SET item_type = CASE WHEN preparation_time > 0 THEN 'service' ELSE 'product' END
WHERE NOT EXISTS (SELECT 1 FROM inventory i WHERE i.product_id = products.id);

ALTER TABLE orders ADD COLUMN IF NOT EXISTS customer_phone VARCHAR(30);
ALTER TABLE order_items ADD COLUMN IF NOT EXISTS unit_cost DECIMAL(10,2) NOT NULL DEFAULT 0;

CREATE INDEX IF NOT EXISTS idx_products_item_type ON products(item_type);
CREATE UNIQUE INDEX IF NOT EXISTS idx_inventory_product_unique ON inventory(product_id);

COMMIT;
