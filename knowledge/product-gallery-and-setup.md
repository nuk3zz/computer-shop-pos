# Product galleries and first-run setup

Source: [`raw/2026-09-28-product-gallery-setup-wizard-request.md`](../raw/2026-09-28-product-gallery-setup-wizard-request.md)

## Confirmed behavior

- A catalog entry supports zero to ten ordered images.
- The first image is the catalog thumbnail and remains mirrored in the legacy `products.image_url` field for compatibility.
- Selecting a thumbnail opens a full-screen gallery with keyboard, button, and touch-swipe navigation.
- A singleton server-side shop profile stores company name, optional logo, and setup completion.
- An authenticated administrator is redirected to setup until the profile is completed.
- Company identity survives browser changes, container rebuilds, and restarts because it is stored in PostgreSQL; uploaded files remain in the persistent uploads volume.

## Data model

- `product_images`: child rows keyed to `products`, ordered from 0 through 9 and deleted with the product.
- `shop_profile`: singleton row with `id = 1`, `company_name`, `logo_url`, and `setup_completed`.

## Compatibility and migration

- Existing primary images are copied into `product_images` at position zero.
- The migration is additive and rerunnable; it does not reset business data.
- Existing clients may continue reading `image_url`; gallery-aware clients use `images`.

## Current boundary

The wizard persists the company identity. Other operational settings still follow their existing storage behavior until a broader server-settings migration is requested.
