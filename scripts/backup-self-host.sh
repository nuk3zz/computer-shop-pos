#!/bin/zsh
set -euo pipefail

umask 077
script_dir="${0:A:h}"
project_dir="${script_dir:h}"
backup_dir="$project_dir/backups/self-host"
docker_bin="/usr/local/bin/docker"
export PATH="/Applications/Docker.app/Contents/Resources/bin:$PATH"

if [[ ! -x "$docker_bin" ]]; then
  docker_bin="/Applications/Docker.app/Contents/Resources/bin/docker"
fi

if ! "$docker_bin" info >/dev/null 2>&1; then
  echo "Docker is not running; backup skipped." >&2
  exit 1
fi

cd "$project_dir"
set -a
source .env
set +a

mkdir -p "$backup_dir"
timestamp=$(date +%Y%m%d-%H%M%S)
database_backup="$backup_dir/database-$timestamp.sql.gz"
uploads_backup="$backup_dir/uploads-$timestamp.tar.gz"

"$docker_bin" compose --env-file .env -f docker-compose.yml exec -T postgres \
  pg_dump -U "$POSTGRES_USER" "$POSTGRES_DB" | gzip -9 > "$database_backup"

tar -czf "$uploads_backup" -C "$project_dir" uploads
shasum -a 256 "$database_backup" "$uploads_backup" > "$backup_dir/checksums-$timestamp.txt"

find "$backup_dir" -type f -mtime +14 -delete
echo "Backup completed: $timestamp"
