-- Destructive one-time conversion requested for the current restaurant demo database.
-- Preserve the administrator account, remove all restaurant/demo business data,
-- and create a clean repair-shop catalog.

BEGIN;

DELETE FROM order_status_history;
DELETE FROM payments;
DELETE FROM order_items;
DELETE FROM orders;
DELETE FROM inventory;
DELETE FROM products;
DELETE FROM categories;
DELETE FROM dining_tables;
DELETE FROM users WHERE username <> 'admin';

ALTER TABLE users DROP CONSTRAINT IF EXISTS users_role_check;
ALTER TABLE users ADD CONSTRAINT users_role_check
    CHECK (role IN ('admin', 'manager', 'sales', 'technician'));

ALTER TABLE orders DROP CONSTRAINT IF EXISTS orders_order_type_check;
ALTER TABLE orders ADD CONSTRAINT orders_order_type_check
    CHECK (order_type IN ('sale', 'service'));

UPDATE users
SET email = 'admin@universal-repair-pos.local',
    first_name = 'Shop',
    last_name = 'Administrator',
    role = 'admin',
    is_active = true
WHERE username = 'admin';

INSERT INTO categories (name, description, color, sort_order) VALUES
('Memory & RAM', 'Desktop and laptop memory modules', '#2563EB', 1),
('Processors', 'Desktop and workstation CPUs', '#7C3AED', 2),
('Motherboards', 'Desktop motherboards and related boards', '#0891B2', 3),
('Storage', 'SSDs, hard drives, and storage accessories', '#059669', 4),
('Peripherals & Cables', 'Keyboards, mice, display cables, and adapters', '#EA580C', 5),
('Software Services', 'Operating-system, driver, and software installation', '#DB2777', 6),
('Repair Services', 'Diagnostics, cleaning, upgrades, and repair labour', '#DC2626', 7);

INSERT INTO products (category_id, name, description, price, sku, preparation_time, sort_order) VALUES
((SELECT id FROM categories WHERE name = 'Memory & RAM'), '8 GB DDR4 Desktop RAM', 'Standard DDR4 desktop memory module', 35.00, 'RAM-DDR4-8GB', 0, 1),
((SELECT id FROM categories WHERE name = 'Memory & RAM'), '16 GB DDR4 Desktop RAM', 'Standard DDR4 desktop memory module', 60.00, 'RAM-DDR4-16GB', 0, 2),
((SELECT id FROM categories WHERE name = 'Processors'), 'Entry-Level Desktop Processor', 'Starter processor placeholder; replace with your stocked model', 120.00, 'CPU-ENTRY-001', 0, 1),
((SELECT id FROM categories WHERE name = 'Motherboards'), 'Micro-ATX Motherboard', 'General Micro-ATX motherboard placeholder', 110.00, 'MB-MATX-001', 0, 1),
((SELECT id FROM categories WHERE name = 'Storage'), '500 GB SATA SSD', '2.5-inch solid-state drive', 55.00, 'SSD-SATA-500', 0, 1),
((SELECT id FROM categories WHERE name = 'Storage'), '1 TB NVMe SSD', 'M.2 NVMe solid-state drive', 95.00, 'SSD-NVME-1TB', 0, 2),
((SELECT id FROM categories WHERE name = 'Peripherals & Cables'), 'USB Keyboard and Mouse Set', 'Wired keyboard and optical mouse bundle', 25.00, 'PER-KM-001', 0, 1),
((SELECT id FROM categories WHERE name = 'Peripherals & Cables'), 'HDMI Cable', 'Standard HDMI display cable', 8.00, 'CAB-HDMI-001', 0, 2),
((SELECT id FROM categories WHERE name = 'Software Services'), 'Windows Installation', 'Operating-system installation and initial setup', 25.00, 'SVC-WINDOWS', 300, 1),
((SELECT id FROM categories WHERE name = 'Software Services'), 'Driver Installation', 'Install and verify required device drivers', 12.00, 'SVC-DRIVERS', 120, 2),
((SELECT id FROM categories WHERE name = 'Repair Services'), 'Computer Diagnostics', 'Hardware and software fault inspection', 15.00, 'SVC-DIAG', 180, 1),
((SELECT id FROM categories WHERE name = 'Repair Services'), 'Desktop Cleaning Service', 'Internal dust cleaning and basic inspection', 20.00, 'SVC-CLEAN', 120, 2),
((SELECT id FROM categories WHERE name = 'Repair Services'), 'Hardware Installation Labour', 'Install or replace a customer-supplied component', 18.00, 'SVC-INSTALL', 90, 3);

INSERT INTO inventory (product_id, current_stock, minimum_stock, maximum_stock, unit_cost)
SELECT
    id,
    CASE WHEN preparation_time = 0 THEN 5 ELSE 999 END,
    CASE WHEN preparation_time = 0 THEN 2 ELSE 0 END,
    CASE WHEN preparation_time = 0 THEN 50 ELSE 999 END,
    CASE WHEN preparation_time = 0 THEN price * 0.70 ELSE 0 END
FROM products;

COMMIT;
