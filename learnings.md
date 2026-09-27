# Learnings

## 2026-09-27 - Fork and discovery

### Worked

- Forked `madebyaris/poinf-of-sales` to `nuk3zz/computer-shop-pos` and configured the source project as the `upstream` remote.
- Confirmed the MIT license permits modification and commercial use while retaining the license.
- The inherited Go + React + PostgreSQL structure already provides authentication, catalog management, orders, payments, reporting, and printable receipt UI.

### Gaps and risks found

- The current product is deeply restaurant-oriented: dining tables, kitchen roles, food seed data, and fixed restaurant status names.
- Inventory cost exists in the database but is not exposed through the product API or admin UI.
- The current sales reports show revenue, not gross profit.
- Product duration is stored as `preparation_time` in minutes and the UI caps it at 120 minutes, which is unsuitable for multi-hour or multi-day repair work.
- Workflow states use hard-coded database checks and TypeScript unions; editable labels need stable internal keys plus configurable display labels.
- Existing float-based Go money fields should not be expanded without a deliberate money-handling pass.

### Direction chosen

- Keep the project brand-neutral under the temporary name “Computer Shop POS.”
- Prioritize computer products and computer services/repairs for V1.
- Reuse the strongest inherited foundations, but replace restaurant assumptions incrementally rather than applying a superficial text-only rename.

### Requirement added during discovery

- Product/service thumbnails are a V1 must-have because the visual grid is intended for fast item recognition.
- The inherited form accepts only a remote image URL; V1 needs a real authenticated upload endpoint, image validation, persistent storage, and an edit-form preview.

## 2026-09-27 - Product image upload foundation

### Worked

- Added an admin-only multipart upload endpoint with content-based JPG, PNG, WebP, and GIF validation and a 5 MB limit.
- Added persistent `uploads/` mounts to both Docker Compose configurations while keeping runtime images out of Git.
- Added image selection, preview, removal, upload-on-save, and fallback behavior to the product form and visual catalog.
- Added backend tests proving that a valid PNG is stored and text disguised with an image filename is rejected.
- Added the missing `go.sum` to make the inherited Dockerfiles' dependency-copy step reproducible.
- The Go test suite passes and Vite produces a production bundle.

### Inherited issues still open

- The full `npm run type-check` now gets past the inherited broken TypeScript project-reference configuration but reports many existing application errors across restaurant-era screens.
- `npm ci` reports 50 dependency advisories, including two critical advisories. Dependency upgrades need a separate compatibility-tested security pass.

## 2026-09-27 - Mac Mini self-hosting

### Worked

- Installed and started Docker Desktop on the Apple Silicon Mac Mini, then runtime-tested the complete frontend, backend, and PostgreSQL stack.
- Changed browser API and image URLs to same-origin paths so other devices on the LAN do not incorrectly call their own `localhost`.
- Published only the Nginx frontend on port 3000; the Go API and PostgreSQL stay private inside the Compose network.
- Added persistent image storage, randomized database/JWT secrets, an owner-only administrator credential file, and automatic disabling of inherited demo accounts.
- Removed public demo passwords and restaurant branding from the login screen.
- Added login-time Docker startup and a containerized daily 02:30 Sri Lanka-time database/uploads backup job with checksums and 14-day local retention.
- Added Docker build ignore files; frontend build context dropped from about 369 MB to about 16 KB.
- Upgraded the frontend build image from Node 18 to Node 22 because an installed dependency requires Node 20.18.1 or newer.

### Issues resolved

- Docker Desktop installed its application bundle before Homebrew could create privileged command-line symlinks. Project scripts now use Docker Desktop's bundled CLI and add its credential-helper directory to `PATH`, avoiding a sudo-dependent installation step.
- macOS LaunchAgents could not read a shell script below the protected Desktop directory. The login agent now opens Docker Desktop directly, while Compose restart policies and an internal backup scheduler handle the application without background shell access to Desktop.
- The inherited production Dockerfile omitted development dependencies even though Vite and TypeScript are required to build. Production builds now install the locked dependency set and run the bundle-only build while the separate inherited type errors remain tracked.

### Operational limits

- The current LAN address can change unless the router reserves an address for the Mac Mini's Ethernet adapter.
- Backups stored on the same Mac protect against application mistakes, not disk loss or theft; real shop use requires a second physical or remote copy.
- At that checkpoint the application still contained restaurant-era data and screens; the later conversion session below supersedes that limitation for the active UI.

## 2026-09-27 - Computer-shop workflow conversion

### Worked

- Replaced the active restaurant navigation and screens with Sales & Services, Repair Tickets, and Catalog & Inventory; legacy restaurant routes now redirect and their unused frontend modules were removed.
- Reset the live demo database to seven computer categories and thirteen starter products/services after creating and checksum-verifying a recoverable PostgreSQL/uploads backup.
- Added explicit product/service type, actual cost, selling price, customer phone, and order-line cost snapshots through a forward database migration.
- Service items now automatically switch checkout to a repair ticket, and the queue tracks Waiting, Diagnosing, In progress, Waiting for customer, and Delivered / completed.
- Added editable WhatsApp message templates and `wa.me` links that include the customer phone and job number without requiring an API integration.
- Defaulted UI currency to LKR, tax and service charge to zero, retained USD as an option, and recorded a future Discord webhook setting without sending webhooks yet.
- Added a responsive tile-size slider and compact square catalog cards; uploaded images now render in the selling grid and cart.
- Completed sales reports now calculate gross profit from the cost captured on each order line; services default to zero cost.
- Removed obsolete restaurant frontend modules and repaired inherited TypeScript issues until both `npm run type-check` and the production Vite build pass.
- Runtime API checks verified service-ticket creation, phone persistence, zero tax, repair status updates, and positive product-profit reporting; temporary verification transactions and accounts were removed.

### Important boundaries

- Settings are currently stored in browser local storage, so another device does not yet share them; server-persisted settings remain a later hardening task.
- The Discord webhook field is a saved placeholder only. No message is transmitted until a dedicated integration is implemented.
- Cost snapshots and profit reporting are implemented, but supplier purchases and atomic stock deductions still require the inventory phase.

## 2026-09-28 - Fresh shop, compact catalog, and reusable clients

### Worked

- Replaced all starter catalog seed data with a blank-business bootstrap that retains only the secured administrator; the live database was backed up and then cleared of categories, products/services, transactions, clients, dining remnants, and non-admin users.
- Added a reusable `customers` table and stable `orders.customer_id` link while retaining the name/phone snapshot on each transaction.
- Checkout now searches saved clients as the operator types, fills the selected phone number, automatically creates or updates a client by unique phone, and requires name plus phone for service tickets.
- Added Manage Clients with name/phone search, transaction totals, and linked product-sale and service/repair history.
- Rebuilt sales cards around a compact 4:3 image, predictable content hierarchy, clamped text, and bottom-aligned controls; the width slider remains available for monitor-specific density.
- Converted the catalog user experience from minutes to whole service days with a one-day minimum. The legacy minute column remains an internal compatibility detail and stores one day as 1,440 minutes.
- Fixed order-history loading for products whose optional description is null.

### Verification

- Verified the database/uploads backup checksum and archive integrity before the destructive reset.
- Go tests, TypeScript checks, and the production Vite build pass.
- A disposable PostgreSQL container confirmed a clean initialization creates exactly one administrator and zero categories, products, clients, or orders.
- Runtime API checks confirmed automatic client saving, stable client-linked service history, and two-day duration persistence; all temporary verification rows were removed afterward.
- Browser inspection confirmed the blank catalog/client states, compact aligned cards across mixed title lengths, and the `Estimated Service Duration (days)` form label.

### Recovery point

- Pre-reset data and uploads can be restored from `backups/self-host/*20260927-215207*`.

## 2026-09-28 - Conditional catalog fields and manual client profiles

### Worked

- Physical-product forms now show Quantity in Stock and hide service duration; service/repair forms show duration in days and hide stock quantity.
- Actual Cost and Selling Price remain available for both catalog types so service material costs can be captured.
- Product API responses now include the persisted inventory quantity, and product plus inventory writes are committed atomically.
- Removed the inherited fake stock value of 999 for services; services carry zero stock.
- Added manual client creation and editing with name, WhatsApp/contact number, optional email, notes, and optional photo.
- Client photos reuse the authenticated, validated 5 MB image-upload flow and render in the client list and profile header.

### Verification

- Go tests, TypeScript checking, and the production frontend build pass.
- Runtime API checks proved a physical product persisted stock 7, updated to 11, and a service retained nonzero material cost plus two-day duration while forcing stock to zero.
- Runtime API checks also proved manual client creation and editing of name, phone, email, notes, and photo reference; only exact temporary verification rows were removed.
- Browser inspection confirmed Physical Product shows Quantity in Stock without duration, Service / Repair shows one-day-minimum duration without stock, and Manage Clients exposes the complete Add Client form.
- A verified pre-migration recovery backup is available at `backups/self-host/*20260927-220817*`.

## 2026-09-28 - Compact sales and catalog cards

### Worked

- Removed forced minimum heights from sales-card title, description, metadata, and action sections.
- Changed sales thumbnails from 4:3 to 16:10, reduced typography and padding, collapsed category/type/stock/duration into one metadata line, and tightened the width range to 130-220 pixels.
- Replaced catalog-card metric pills with a plain two-column operational summary and changed decorative gradient placeholders to neutral inventory styling.
- Reduced catalog search-container padding, card gaps, thumbnail size, and edit/delete controls while retaining all price, cost, profit, stock/duration, category, and type information.

### Verification

- TypeScript checking and the production Vite build pass.
- Browser inspection against the real H81 item confirmed a substantially shorter sales tile and a compact catalog card with no large internal blank area.
