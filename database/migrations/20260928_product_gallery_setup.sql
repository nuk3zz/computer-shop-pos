BEGIN;

CREATE TABLE IF NOT EXISTS product_images (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    product_id UUID NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    image_url VARCHAR(500) NOT NULL,
    sort_order INTEGER NOT NULL DEFAULT 0 CHECK (sort_order >= 0 AND sort_order < 10),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (product_id, sort_order)
);

CREATE INDEX IF NOT EXISTS idx_product_images_product_id ON product_images(product_id);

INSERT INTO product_images (product_id, image_url, sort_order)
SELECT id, image_url, 0
FROM products
WHERE image_url IS NOT NULL AND image_url <> ''
ON CONFLICT (product_id, sort_order) DO NOTHING;

CREATE TABLE IF NOT EXISTS shop_profile (
    id SMALLINT PRIMARY KEY DEFAULT 1 CHECK (id = 1),
    company_name VARCHAR(150) NOT NULL DEFAULT 'Computer Shop POS',
    logo_url VARCHAR(500),
    setup_completed BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

INSERT INTO shop_profile (id, company_name, setup_completed)
VALUES (1, 'Computer Shop POS', false)
ON CONFLICT (id) DO NOTHING;

COMMIT;
