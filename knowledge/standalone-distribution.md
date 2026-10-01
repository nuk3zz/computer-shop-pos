# Standalone desktop/server distribution

Source: [`raw/2026-09-28-standalone-installers-backup-update-request.md`](../raw/2026-09-28-standalone-installers-backup-update-request.md)

## Goals

- Windows x64, Linux x64, and universal macOS installation without Docker.
- One lightweight Go server with the compiled web interface embedded.
- SQLite for native installations; PostgreSQL remains supported for the existing Docker deployment.
- Durable state outside the installation directory so upgrades replace program files without replacing shop data.

## Default data layout

Windows/Linux: `Documents/Universal Repair POS/`.

macOS: `~/Library/Application Support/Universal Repair POS/`. On first upgraded launch, an existing Documents installation is moved as a whole; a failed move stops startup to prevent an empty shop from replacing existing records. Once the destination exists, startup never inspects Documents. Explicit data-directory overrides remain supported.

- `data/universal-repair-pos.db`: SQLite database.
- `uploads/`: product, logo, and client images.
- `backups/`: automatic and manual backup archives.
- `config/`: installation identity and JWT secret.
- `logs/`: native runtime logs where the service wrapper supports them.

An explicit `UNIVERSAL_REPAIR_POS_DATA_DIR` environment value overrides this location for administrators and package testing.

## First-run security

- Fresh native databases contain only a disabled bootstrap row.
- Setup completion is accepted from the same computer, or from a private-LAN device for a headless Linux installation, only while setup is incomplete. PostgreSQL/Docker installations retain the stricter local-or-authenticated setup path.
- Setup requires the owner to choose a username and password and activates the administrator atomically with the company profile.
- Normal authentication begins only after setup completion.

## Backups and restore

- Native mode creates a consistent SQLite snapshot and packages it with uploads and a manifest.
- A daily scheduler creates at most one automatic backup per local calendar day and catches up after startup if the scheduled time was missed.
- Manual backups use the same verified archive format.
- Restore validates the archive before staging it and makes a safety backup first.
- Restores and upgrades never write business data into the application installation directory.

## Updates

Source: [`raw/2026-09-29-built-in-updater-and-macos-port-conflict.md`](../raw/2026-09-29-built-in-updater-and-macos-port-conflict.md)

- The application reports its build version and checks the project's GitHub Releases feed on demand.
- Before an update, the standalone application creates a verified safety backup, downloads the matching release asset, and verifies its GitHub-published SHA-256 digest.
- macOS and Windows open the verified installer for the owner to approve. Headless Linux displays the exact `sudo apt install <path>` command after downloading and verifying the `.deb`.
- The Linux package requires the system CA certificate bundle so update traffic remains TLS-verified on minimal server installations.
- Windows and Linux packages replace only application files. The Documents data directory is intentionally excluded from package ownership.
- Database migrations are versioned, additive, and applied at startup before serving requests.

## Packages

- The Windows x64 Inno Setup package permits an installation-path choice, creates startup and browser shortcuts, and can add a TCP 3000 rule limited to Windows' private-network profile.
- The Linux x64 Debian package configures a systemd service owned by the invoking non-root user, stores data under that user's Documents directory, and provides `universal-repair-pos-status` to print service health and detected URLs.
- The universal macOS package installs Apple Silicon and Intel code plus a per-user LaunchAgent, stores data in Application Support, and starts at sign-in. Postinstall preserves a user-local launcher override when present.
- Package uninstall metadata never owns or removes the Documents data directory.

## macOS migration and updates

Source: [`raw/2026-09-29-macos-first-run-port-conflict.md`](../raw/2026-09-29-macos-first-run-port-conflict.md)

- Do not run the Docker and native macOS editions on port 3000 simultaneously. A browser can otherwise reach the older Docker login while the fresh native SQLite installation is still waiting at `/setup`.
- When moving this Mac from Docker to the native package, stop the Docker Compose stack without removing its volumes, then restart the native LaunchAgent and open the native address (`http://localhost:3210` beginning with v0.3.4).
- Native macOS updates are installed by running the newer `.pkg` over the existing installation. The package replaces the binary and LaunchAgent only; shop data remains intact, with legacy Documents storage migrated to Application Support.
- The in-app updater selects the installer matching the running operating system, creates a verified safety backup, downloads the installer into the persistent updates folder, and verifies the SHA-256 digest published with the GitHub Release.
- On macOS and Windows, **Download & Install** opens the verified local installer; the owner completes the normal Apple installer or Windows UAC prompt. The updater never attempts to bypass operating-system authorization.
- Repeated update clicks reuse the existing installer only when both its byte size and SHA-256 digest match the release metadata. This keeps the installer path immutable while macOS Installer is waiting for administrator approval; replacing an open package causes macOS to reject it as changed.
- A headless Linux server downloads and verifies the `.deb`, then displays the exact `sudo apt install <path>` command for the administrator to run.
- macOS uses `http://localhost:3210` beginning with v0.3.4 to avoid collisions with development and Adobe extension processes that commonly claim port 3000. Explicit data-directory overrides are preserved.

## Network modes

- `local`: requests are accepted only from loopback addresses.
- `lan`: devices on the same trusted network may connect to the host's displayed address.
- The application never configures router port forwarding and must not be exposed directly to the public internet.
