# Universal Repair POS

A universal sales, inventory, service-workflow, and invoicing system built from the open-source [`madebyaris/poinf-of-sales`](https://github.com/madebyaris/poinf-of-sales) project.

**Universal Repair POS** is designed for repair-and-retail businesses such as computer, phone, electronics, appliance, and general service shops. Each installation uses the owner's own company name and logo.

## V1 direction

- Products and services in one visual catalog.
- Uploaded thumbnails for fast item recognition.
- Supplier cost, selling price, stock, and low-stock visibility.
- Sales and repair/work orders containing products, services, or both.
- Trackable service tickets with customer phone details and WhatsApp status messages.
- Revenue, cost, gross-profit, and margin reports.
- Basic printable invoices, with a custom template planned later.

The confirmed scope and staged implementation plan are in the [product brief](knowledge/product-brief.md) and [V1 roadmap](knowledge/v1-roadmap.md).

## Current foundation

- React 18, TypeScript, Vite, Tailwind CSS, and shadcn/ui.
- Go and Gin REST API.
- SQLite in standalone Windows, Linux, and macOS installations; PostgreSQL remains supported by Docker.
- JWT authentication and role-based access.
- Docker Compose development and production definitions.

The active navigation and workflows no longer expose restaurant tables, servers, kitchen screens, or food demo data. Legacy database compatibility structures remain internal so existing installations and backups continue working after upgrades.

## Docker self-hosting

The Docker stack is available on its host at `http://localhost:3000` and can be made available to trusted devices on the same local network at `http://<server-ip>:3000`. Only the web entry point is exposed; PostgreSQL and the backend API stay private inside Docker.

See the [self-hosting guide](knowledge/self-hosting.md) for startup, backup, and safety details.

## Standalone installers

The [GitHub Releases page](https://github.com/nuk3zz/universal-repair-pos/releases) provides Windows, Linux, and macOS packages. None requires Docker, PostgreSQL, Node.js, or Go on the shop computer.

### iPad Home Screen app

An iPad uses the interface from a running POS server; it does not store the main database itself.

1. In the server's setup or **Settings → Server Access**, select **Same Wi-Fi / LAN**.
2. Connect the iPad to the same trusted network and open the server's displayed LAN address in Safari. Do not use `localhost` on the iPad.
3. In Safari, tap **Share → Add to Home Screen**, enable **Open as Web App**, and tap **Add**.

The Home Screen icon opens in its own app-style window and receives application updates from the server automatically. The server must remain running. Internet use outside the shop network requires a properly secured HTTPS deployment; do not expose port 3000 directly through the router.

### Windows 64-bit

1. Download `Universal-Repair-POS-<version>-Windows-x64-Setup.exe`.
2. Run the installer and choose an installation folder. The default is under `C:\Program Files`.
3. Leave the private-LAN firewall option enabled if phones or laptops on the same trusted Wi-Fi should connect.
4. The installer starts the server and opens `http://localhost:3000/setup`.
5. Enter the company identity and create the owner username and password.

Windows starts the server automatically when the PC starts. Shop data is kept separately in `Documents\Universal Repair POS`, so installing a newer setup file over the current version does not replace the database, photos, or backups.

### Linux x64 headless server

On the mini PC, download the `.deb` file and run:

```bash
sudo apt install ./universal-repair-pos_<version>_amd64.deb
universal-repair-pos-status
```

The installer enables a lightweight systemd service and prints both `http://localhost:3000` and the detected `http://192.168.x.x:3000` addresses. Open the LAN address from a laptop or phone on the same trusted network, finish owner setup, and select **Same Wi-Fi / LAN**. Shop data is stored under the installing user's `Documents/Universal Repair POS` folder.

Useful Linux commands:

```bash
universal-repair-pos-status
sudo systemctl restart universal-repair-pos
sudo journalctl -u universal-repair-pos -n 100 --no-pager
```

### macOS universal

1. Download `Universal-Repair-POS-<version>-macOS-Universal.pkg`.
2. Open the package and complete installation. It installs a background server that starts automatically at sign-in and opens `http://localhost:3000/setup`.
3. Complete owner setup. Choose **Same Wi-Fi / LAN** if an iPad or another trusted local device should connect.

The package supports Apple Silicon and Intel Macs. Shop data remains under `Documents/Universal Repair POS`, separate from application files. Current public packages are not yet Developer ID-signed or notarized, so macOS may block the first opening; production-grade one-click distribution requires Apple signing credentials and notarization.

### Backups, restore, and upgrades

In **Settings → Backup & Restore**, the standalone edition can create or download a manual `.urposbackup`, upload one from another computer, and stage a verified restore. Automatic daily backups default to 02:30 and retain the latest 30 automatic archives. After staging a restore, restart the Windows PC/application, run `sudo systemctl restart universal-repair-pos` on Linux, or sign out and back in on macOS.

The **Check for updates** button reads the latest GitHub Release. Downloading and running a newer installer upgrades the program while retaining the Documents data folder. A same-disk backup helps with accidental changes; important shop data should also be copied to another drive or remote location.

## Development setup

Prerequisites: Docker Desktop with Docker Compose and Make.

```bash
make dev
```

Then open:

- Frontend: `http://localhost:3000`
- Backend health check: `http://localhost:8080/health`
- API base: `http://localhost:8080/api/v1`

Uploaded product images are stored in the ignored local `uploads/` directory and mounted into the backend container. Back up this directory together with PostgreSQL data.

## Project knowledge

- [`CODEX.md`](CODEX.md) - development and knowledge-maintenance rules.
- [`knowledge/`](knowledge/) - product, domain, architecture, and roadmap documentation.
- [`learnings.md`](learnings.md) - dated engineering findings and decisions.
- [`raw/`](raw/) - untouched source material awaiting processing.

## Upstream and license

This repository is a GitHub fork and keeps `madebyaris/poinf-of-sales` configured as the `upstream` Git remote. The original work and this fork are distributed under the [MIT License](LICENSE).
