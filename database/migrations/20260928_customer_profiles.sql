-- Optional customer profile photos for the manually managed client directory.
BEGIN;

ALTER TABLE customers ADD COLUMN IF NOT EXISTS image_url VARCHAR(500);

COMMIT;
