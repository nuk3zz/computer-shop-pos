-- Blank Computer Shop POS bootstrap data.
-- Business data is intentionally empty so every catalog item, client, and staff
-- account represents the real shop rather than sample content.

-- The self-host startup script replaces this bootstrap password immediately.
INSERT INTO users (username, email, password_hash, first_name, last_name, role) VALUES
('admin', 'admin@computer-shop.local', '$2a$10$FPH.ONfAgquWmXjM3LE61OIgOPgXX8i.jOISCHZ2DpK2gg4krEWfO', 'Shop', 'Administrator', 'admin');

INSERT INTO shop_profile (id, company_name, setup_completed)
VALUES (1, 'Computer Shop POS', false);
