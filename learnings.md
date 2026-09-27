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
- The application remains an early computer-shop conversion with restaurant-era data and screens still to replace before production business use.
