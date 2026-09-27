# Standalone desktop/server distribution

Source: [`raw/2026-09-28-standalone-installers-backup-update-request.md`](../raw/2026-09-28-standalone-installers-backup-update-request.md)

## Goals

- Windows x64 and Linux x64 installation without Docker.
- One lightweight Go server with the compiled web interface embedded.
- SQLite for native installations; PostgreSQL remains supported for the existing Docker deployment.
- Durable state outside the installation directory so upgrades replace program files without replacing shop data.

## Default data layout

`Documents/Computer Shop POS/`

- `data/computer-shop-pos.db`: SQLite database.
- `uploads/`: product, logo, and client images.
- `backups/`: automatic and manual backup archives.
- `config/`: installation identity and JWT secret.
- `logs/`: native runtime logs where the service wrapper supports them.

An explicit `COMPUTER_SHOP_DATA_DIR` environment value overrides this location for administrators and package testing.

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

- The application reports its build version and checks the project's GitHub Releases feed on demand.
- Windows and Linux packages replace only application files. The Documents data directory is intentionally excluded from package ownership.
- Database migrations are versioned, additive, and applied at startup before serving requests.

## Packages

- The Windows x64 Inno Setup package permits an installation-path choice, creates startup and browser shortcuts, and can add a TCP 3000 rule limited to Windows' private-network profile.
- The Linux x64 Debian package configures a systemd service owned by the invoking non-root user, stores data under that user's Documents directory, and provides `computer-shop-pos-status` to print service health and detected URLs.
- Package uninstall metadata never owns or removes the Documents data directory.

## Network modes

- `local`: requests are accepted only from loopback addresses.
- `lan`: devices on the same trusted network may connect to the host's displayed address.
- The application never configures router port forwarding and must not be exposed directly to the public internet.
