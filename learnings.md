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

## 2026-09-27 - Docker self-hosting

### Worked

- Installed and started Docker Desktop on the local server, then runtime-tested the complete frontend, backend, and PostgreSQL stack.
- Changed browser API and image URLs to same-origin paths so other devices on the LAN do not incorrectly call their own `localhost`.
- Published only the Nginx frontend on port 3000; the Go API and PostgreSQL stay private inside the Compose network.
- Added persistent image storage, randomized database/JWT secrets, an owner-only administrator credential file, and automatic disabling of inherited demo accounts.
- Removed public demo passwords and restaurant branding from the login screen.
- Added login-time Docker startup and a containerized scheduled database/uploads backup job with checksums and 14-day local retention.
- Added Docker build ignore files; frontend build context dropped from about 369 MB to about 16 KB.
- Upgraded the frontend build image from Node 18 to Node 22 because an installed dependency requires Node 20.18.1 or newer.

### Issues resolved

- Docker Desktop installed its application bundle before Homebrew could create privileged command-line symlinks. Project scripts now use Docker Desktop's bundled CLI and add its credential-helper directory to `PATH`, avoiding a sudo-dependent installation step.
- macOS LaunchAgents could not read a shell script below the protected Desktop directory. The login agent now opens Docker Desktop directly, while Compose restart policies and an internal backup scheduler handle the application without background shell access to Desktop.
- The inherited production Dockerfile omitted development dependencies even though Vite and TypeScript are required to build. Production builds now install the locked dependency set and run the bundle-only build while the separate inherited type errors remain tracked.

### Operational limits

- The current LAN address can change unless the router reserves an address for the server's network adapter.
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

## 2026-09-28 - Product galleries and persistent first-run setup

### Worked

- Added ordered product/service galleries with up to ten photos while retaining `products.image_url` as the backward-compatible cover image.
- Existing product thumbnails migrate into gallery position zero without resetting catalog, inventory, customer, or transaction data.
- Catalog thumbnails now open a full-screen viewer with previous/next controls, keyboard arrows, Escape close, thumbnail selection, and touch-swipe navigation.
- Added a one-time authenticated setup wizard for company name and optional logo; the singleton shop profile is stored in PostgreSQL and logo files use the persistent uploads volume.
- The configured company name and logo render in the admin sidebar after setup.

### Verification

- Go tests, TypeScript checking, and the production Vite and Docker builds pass.
- A clean PostgreSQL initialization produced one administrator, zero products/images, and one incomplete shop profile.
- A checksum-verified recovery backup was created at `backups/self-host/*20260928-023054*` before the live migration.
- Runtime API checks proved three-image creation, ordered two-image replacement, legacy cover synchronization, ten-image limit enforcement, persistent shop-profile update/reload, and cleanup of the temporary catalog item.
- Browser inspection confirmed the existing H81 thumbnail opens full screen and the incomplete live profile redirects to the setup wizard.

## 2026-09-28 - Native installers, owner setup, maintenance, and item details

### Worked

- Added a pure-Go SQLite standalone mode with the compiled React interface embedded into one server binary; the existing Docker/PostgreSQL edition remains supported.
- Fresh standalone installations create a disabled bootstrap administrator and require a two-step wizard for company identity, local/LAN choice, owner name, username, and password before login is possible.
- Headless Linux setup permits the one-time owner wizard from a private-LAN device; after setup the selected local-only or LAN policy is enforced using the direct socket address rather than spoofable proxy headers.
- Added verified `.cspbackup` archives containing a consistent SQLite snapshot, uploads, manifest, and checksums; manual backup/download/upload, staged restore, pre-restore safety backup, daily scheduling, and 30-archive automatic retention are exposed in Settings.
- Added Windows x64 Inno Setup and Linux x64 Debian/systemd packaging. Application upgrades replace program files while the database, uploads, logs, and backups remain under `Documents/Computer Shop POS`.
- Added an on-demand GitHub Release update check and a release workflow that builds both installers for version tags.
- Replaced the photo-only overlay with a responsive item-detail viewer that retains ten-photo navigation and shows the complete description/compatibility text, category, price, SKU, stock, or service duration. Items without photos now open a compact detail view.
- Applied the additive network/backup preference migration to the running Mac Docker database and redeployed the current frontend/backend after a verified recovery backup.

### Verification

- Go formatting and tests, TypeScript checking, the Vite production build, Actionlint, ShellCheck, and Git whitespace checks pass.
- Cross-compilation produced a static Linux x64 binary and a Windows x64 GUI executable; a Debian package was built and its control metadata inspected.
- An isolated native runtime proved fresh database initialization, owner setup/login, product description persistence, portable weekly/monthly reports, manual backup verification, restore staging, service restart, and rollback of data created after the selected backup.
- Browser inspection confirmed the standalone Backup & Restore controls and the compact product detail viewer with multiline compatibility/features content.
- The live PostgreSQL installation retained its catalog and profile data, and authenticated catalog, report, and system-information smoke tests passed after deployment.

### Recovery point

- The pre-migration live Docker database and uploads are available at `backups/self-host/*20260928-032026*`.

## 2026-09-28 - Persistent identity, owner profile, and guarded fresh start

- The sidebar read `shop_profile` while Settings wrote the shop name only to browser storage, so a successful-looking save could never update the visible identity. Shop name, logo, and sidebar description now share the server profile and React Query cache.
- Account-menu items without navigation silently looked broken. The Profile item now opens a real authenticated editor, and saving replaces both the server row and cached `pos_user` value so the sidebar refreshes immediately.
- Existing native SQLite databases need explicit `PRAGMA table_info` checks before additive `ALTER TABLE` statements; changing only `CREATE TABLE IF NOT EXISTS` does not upgrade installed databases.
- Destructive reset is safest as one transaction with dependency-ordered deletes. The owner account, network/backup preferences, and backup archives remain outside the reset boundary, while standalone mode creates a verified safety backup before deletion.
- The legacy `scripts/backup.sh` assumes a `postgres` database role and failed against the hardened `pos_app` deployment. The active `pos-backup` container script produced the verified pre-deployment recovery point instead.

## 2026-09-28 - Public documentation privacy correction

- A local deployment detail was incorrectly promoted into the public README and knowledge pages. A development machine is operational context, not product identity.
- Public examples now use generic server/owner placeholders, and `CODEX.md` requires a personal-identifier scan before documentation commits.

## 2026-09-28 - iPad Home Screen app and macOS packaging

- The lowest-maintenance iPad distribution is the existing responsive client installed from Safari with Open as Web App. It still needs a reachable server and intentionally does not claim offline sales or inventory support.
- Added explicit web-app metadata and a dedicated Home Screen icon instead of the inherited Vite favicon.
- A macOS package can reuse the pure-Go native server: two Go builds combine into one universal binary, while a LaunchAgent provides automatic startup and keeps SQLite data outside package ownership.
- Unsigned public macOS packages are useful for internal testing but are not equivalent to Developer ID signing and Apple notarization; those require owner-held Apple credentials.

## 2026-09-28 - Universal Repair POS identity

- Adopted **Universal Repair POS** as the canonical product name so the application can serve computer, phone, electronics, appliance, craft, and other repair-and-retail businesses without category-specific branding.
- Renamed public application surfaces, package artifacts, native binaries, services, launchers, the Home Screen manifest, updater endpoint, documentation, and the repository slug.
- A product rename must not look like a fresh installation: the standalone runtime and installers migrate the former Documents directory and SQLite filename while continuing to accept legacy backup archives and environment configuration.
- Windows keeps its stable Inno Setup application identifier for in-place upgrades; Linux declares package replacement and macOS retires the former LaunchAgent after preserving data.

## 2026-09-29 - Staff password update repair

- Zod's `.optional()` permits `undefined`, not an empty string supplied by a controlled form. Optional edit passwords require an explicit blank-or-valid refinement before the submit handler can omit blank values.
- The staff update client used `PATCH` while the backend registered `PUT`, causing every otherwise-valid update to return 404. Client methods must match the route contract exactly.
- Password length is now checked server-side as well as in the form so direct API requests cannot set an undersized replacement.
- The owner account itself was still active and its original owner-only saved credential continued to authenticate, showing that the earlier 404 had not changed its password. Account diagnosis should test credentials without printing them.
- Login tokens already persisted across browser restarts in local storage; an explicit default-on 30-day option now makes the duration understandable while invalid credentials remain deliberately generic to prevent username discovery.
- Verify the production path through the frontend reverse proxy (`/health` on port 3000); the backend container is intentionally not exposed directly on host port 8080.

## 2026-09-29 - macOS native first-run port conflict

- A fresh native SQLite installation can correctly report `setup_completed = false` while Safari shows an existing Docker installation's login page if both editions listen on port 3000.
- Diagnose first-run routing by comparing the native database, `/api/v1/setup/status`, active listeners, Docker Compose state, and the LaunchAgent log before resetting any data.
- Stopping the old Docker stack (without deleting volumes), restarting the native LaunchAgent, and loading `/setup` resolved the conflict; the wizard was then verified through Safari's accessibility tree.
- macOS package updates are in-place: reinstalling a newer `.pkg` replaces program files while preserving `Documents/Universal Repair POS/`.

## 2026-09-29 - Atomic product sales, supplier debt, and stock labels

- A successful order row alone is not a completed sale. Product checkout must atomically save/link the client, snapshot selling and cost prices, decrement stock, create fulfillment state, and record any payment already received.
- Financial reporting must distinguish customer cash collected from earnings: LKR 15,000 collected on an item costing LKR 10,000 is LKR 15,000 sales collected and LKR 5,000 gross profit.
- SQLite aggregate timestamps can be returned as strings. Scanning `MAX(created_at)` directly into `*time.Time` made the entire client list fail after its first order; parse nullable aggregate values explicitly.
- COD fulfillment and payment are separate facts. A COD order completes only after both delivery and full payment; prepaid delivery records money immediately but remains operationally open until delivered.
- Supplier purchases and payments work best as separate append-only records. Inventory and acquisition cost update with the purchase transaction, while outstanding debt is total purchases minus all supplier payments.
- Non-credit suppliers must reject partial payment before inventory changes. Runtime verification confirmed the rejected request left stock unchanged.
- Additive SQLite indexes that reference new columns must be created after compatibility `ALTER TABLE` steps. Putting the index in the base schema prevents older installed databases from starting before migration runs.
- Stock state is clearest as unboxed text: green In stock, red Out of stock, and amber Pre-order. Pre-order uses an explicit catalog flag; it is not guessed from a zero count.

### Verification and recovery

- Backend tests, TypeScript checking, and the production frontend build pass.
- A disposable native SQLite runtime proved stock decrement, client persistence, LKR 15,000 sales collected, LKR 5,000 gross profit, explicit pre-order persistence, supplier stock receipt, debt repayment, COD completion, and non-credit rollback.
- A copy of a pre-feature native database started successfully and gained the fulfillment, stock-commit, and pre-order columns through additive migration.
- Before repairing the one known live sale, a checksum-verified recovery set was saved under `Documents/Universal Repair POS/backups/recovery-20260929-150610`; the exact order was then completed, paid, and its stock changed from one to zero.

## 2026-09-29 - Built-in updates, dashboard links, and supplier evidence

- A macOS installer can succeed while its LaunchAgent remains unloaded; post-install must retry `launchctl bootstrap`, kick-start the service, wait for a real health response, and leave a diagnostic log instead of silently opening a dead URL.
- Port 3000 was also occupied by an Adobe CEP extension on this machine. The macOS package now uses port 3210 while Windows, Linux, and Docker retain port 3000, and the system-information response derives its localhost port from the active request.
- The built-in updater creates a safety backup, selects the operating-system-specific GitHub Release asset, limits its size, verifies the release SHA-256 digest, and then opens the macOS/Windows installer or returns an explicit Linux `sudo apt install` command. Semantic comparison prevents an older release from being offered as an update.
- Minimal Linux systems need the CA certificate bundle for TLS-verified GitHub update checks; the Debian package now declares it.
- A card with only `cursor-pointer` is not an action. Dashboard shortcut cards and header controls now use real TanStack Router links, which were click-tested in an isolated v0.3.4 native build.
- Dashboard financial color semantics are purple for customer money collected, red for product/service cost, and green for gross profit.
- Opening stock can be linked to a supplier during catalog creation by creating the catalog row at zero stock and then using the supplier-purchase path to add stock once. On purchase failure, the newly created catalog row is removed so a misleading unlinked item is not left behind.
- Supplier purchase summaries describe the payment made at purchase time; later unallocated supplier-level payments must not be presented as if they changed that historical initial-payment field. A separate append-only transaction history now shows the purchase and every payment with its own timestamp.
- Optional supplier evidence accepts inspected PDF/JPG/PNG/WebP files up to 10 MB. The database stores only application-owned `/uploads/supplier-documents/` paths, and native backup/restore already includes the uploads tree.
