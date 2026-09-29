# Settings, owner profile, and fresh start

Source: [`raw/2026-09-28-settings-profile-start-fresh-request.md`](../raw/2026-09-28-settings-profile-start-fresh-request.md)

## Persistent shop identity

- `shop_profile` is the authoritative source for company name, logo, and the short sidebar description.
- Saving Shop Information updates the server and the visible sidebar immediately; it is not a browser-only preference.
- Currency, receipt copy, webhook, and WhatsApp templates remain device-local until their broader server-side settings model is introduced.

## Owner profile

- The Profile menu opens an authenticated editor for first name, last name, username, email, and an optional profile photo.
- A saved profile replaces the cached signed-in user so the sidebar reflects the change without requiring a new login.

## Start Fresh

- Start Fresh is administrator-only and requires typing `START FRESH`.
- It deletes catalog entries, categories, inventory, sales, repair tickets, payments, clients, legacy tables, and uploaded media.
- It resets company identity and returns the installation to the setup wizard.
- The current owner account, network mode, automatic-backup preference, and existing backup archives are retained so the owner is not locked out and can recover if needed.

Related: [product gallery and setup](product-gallery-and-setup.md), [clients and fresh-start behavior](clients-and-fresh-start.md), and [standalone distribution](standalone-distribution.md).

## Staff password edits

Source: [`raw/2026-09-29-staff-password-update-bug.md`](../raw/2026-09-29-staff-password-update-bug.md)

- The staff edit form treats a blank or whitespace-only new-password value as “keep the existing password.”
- A nonblank replacement remains subject to the six-character minimum in both the frontend and backend.
- The staff update client uses `PUT /api/v1/admin/users/:id`, matching the registered backend route.

## Login persistence and feedback

- Invalid credentials produce the generic message “Username or password is incorrect.” This is intentionally not field-specific because revealing whether a username exists would weaken account security.
- **Keep me signed in for 30 days** is enabled by default. The signed token is kept in browser local storage, so ordinary browser or device restarts do not require another login during that period.
- Clearing browser site data, explicitly logging out, changing the server's JWT secret, or reaching token expiry requires a new login.
