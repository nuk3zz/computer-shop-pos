# Mac Mini self-hosting

## Address

- Mac Mini: `http://localhost:3000`
- Trusted home-network devices: `http://<mac-mini-ip>:3000`

The network address may change if the router gives the Mac Mini a different DHCP address. Create a router-side DHCP reservation for the Mac Mini's Ethernet adapter before relying on a bookmarked address.

## Runtime

Docker Compose runs four containers:

- `pos-frontend`: Nginx web UI and same-origin proxy on port 3000.
- `pos-backend`: private Go API, reachable only inside the Docker network.
- `pos-postgres`: private PostgreSQL database, reachable only inside the Docker network.
- `pos-backup`: private scheduled backup worker.

Only port 3000 is published to the home network. Do not forward this port through the router or expose it directly to the public internet.

## Persistent data

- PostgreSQL data: Docker volume `computer-shop_postgres_data` (actual prefix may reflect the Compose project directory).
- Uploaded item images: `uploads/` in the project directory.
- Local backup archives: `backups/self-host/`.
- Generated deployment and login secrets: `.env` and `.secrets/`; both are excluded from Git.

## Administrator login

- Username: `admin`
- Password: stored locally in `.secrets/admin-login.txt` with owner-only permissions.
- `scripts/secure-first-admin.sh` creates the password on first start, reapplies its password hash on later starts, and disables the inherited demo accounts.

Never paste the password into Git, documentation, screenshots, or support messages.

## Automatic operation

- `scripts/start-self-host.sh` opens Docker Desktop when necessary, starts the stack, and secures the administrator account.
- A login LaunchAgent opens Docker Desktop when the Mac user signs in. It does not read the Desktop project folder, avoiding macOS background-process privacy restrictions.
- The versioned LaunchAgent definition is `deploy/macos/com.nuk3zz.computer-shop-pos.start.plist`; its installed copy belongs in `~/Library/LaunchAgents/`.
- The `pos-backup` container creates a database dump and uploads archive every day at 02:30 Sri Lanka time (21:00 UTC).
- Containers use `restart: unless-stopped`, so Docker restarts the application and backup worker after Docker Desktop restarts.

Local backups on the same Mac do not protect against disk failure or theft. Copy periodic backups to another physical device before using the system for real shop records.

## Manual commands

```bash
./scripts/start-self-host.sh
docker compose --env-file .env -f docker-compose.yml ps
docker compose --env-file .env -f docker-compose.yml logs --tail=100
./scripts/backup-self-host.sh
docker compose --env-file .env -f docker-compose.yml exec backup /usr/local/bin/backup-container.sh
docker compose --env-file .env -f docker-compose.yml down
```

`down` stops the application but retains database and uploaded-image data. Never add `--volumes` unless intentionally deleting the database after a verified backup.
