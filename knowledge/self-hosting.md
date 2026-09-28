# Docker self-hosting

## Address

- Server computer: `http://localhost:3000`
- Trusted local-network devices: `http://<server-ip>:3000`

The network address may change if the router gives the server a different DHCP address. Create a router-side DHCP reservation for the server's network adapter before relying on a bookmarked address.

## Runtime

Docker Compose runs four containers:

- `pos-frontend`: Nginx web UI and same-origin proxy on port 3000.
- `pos-backend`: private Go API, reachable only inside the Docker network.
- `pos-postgres`: private PostgreSQL database, reachable only inside the Docker network.
- `pos-backup`: private scheduled backup worker.

Only port 3000 is published to the local network. Do not forward this port through the router or expose it directly to the public internet.

## Persistent data

- PostgreSQL data: the Compose-managed `postgres_data` volume (its actual prefix reflects the local project directory).
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
- On macOS, a login LaunchAgent can open Docker Desktop when the server user signs in. It does not read the project folder, avoiding background-process privacy restrictions.
- The versioned LaunchAgent definition is `deploy/macos/com.nuk3zz.universal-repair-pos.start.plist`; its installed copy belongs in `~/Library/LaunchAgents/`.
- The `pos-backup` container creates a database dump and uploads archive on its configured daily schedule.
- Containers use `restart: unless-stopped`, so Docker restarts the application and backup worker after Docker Desktop restarts.

Local backups on the same server do not protect against disk failure or theft. Copy periodic backups to another physical device before using the system for real shop records.

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
