ALTER TABLE users
    ADD COLUMN IF NOT EXISTS profile_image_url VARCHAR(500);

ALTER TABLE shop_profile
    ADD COLUMN IF NOT EXISTS description VARCHAR(200) NOT NULL DEFAULT 'Sales and repair management';
