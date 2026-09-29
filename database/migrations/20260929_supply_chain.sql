BEGIN;

CREATE TABLE IF NOT EXISTS suppliers (id UUID PRIMARY KEY, name VARCHAR(150) NOT NULL, phone VARCHAR(30), location TEXT, notes TEXT, credit_allowed BOOLEAN NOT NULL DEFAULT false, created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP, updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP);
CREATE TABLE IF NOT EXISTS supplier_purchases (id UUID PRIMARY KEY, supplier_id UUID NOT NULL REFERENCES suppliers(id) ON DELETE RESTRICT, reference_number VARCHAR(100), total_amount NUMERIC(12,2) NOT NULL DEFAULT 0, notes TEXT, purchased_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP, created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP);
CREATE TABLE IF NOT EXISTS supplier_purchase_items (id UUID PRIMARY KEY, purchase_id UUID NOT NULL REFERENCES supplier_purchases(id) ON DELETE CASCADE, product_id UUID NOT NULL REFERENCES products(id) ON DELETE RESTRICT, quantity INTEGER NOT NULL CHECK (quantity > 0), unit_cost NUMERIC(12,2) NOT NULL CHECK (unit_cost >= 0), total_cost NUMERIC(12,2) NOT NULL CHECK (total_cost >= 0));
CREATE TABLE IF NOT EXISTS supplier_payments (id UUID PRIMARY KEY, supplier_id UUID NOT NULL REFERENCES suppliers(id) ON DELETE RESTRICT, purchase_id UUID REFERENCES supplier_purchases(id) ON DELETE SET NULL, amount NUMERIC(12,2) NOT NULL CHECK (amount > 0), notes TEXT, paid_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP, created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP);
CREATE INDEX IF NOT EXISTS idx_supplier_purchases_supplier ON supplier_purchases(supplier_id, purchased_at);
CREATE INDEX IF NOT EXISTS idx_supplier_payments_supplier ON supplier_payments(supplier_id, paid_at);

ALTER TABLE supplier_purchases ADD COLUMN IF NOT EXISTS attachment_url TEXT;
ALTER TABLE supplier_payments ADD COLUMN IF NOT EXISTS attachment_url TEXT;

COMMIT;
