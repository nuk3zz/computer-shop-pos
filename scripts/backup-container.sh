#!/bin/sh
set -eu

backup_dir="/backups"
timestamp=$(date -u +%Y%m%d-%H%M%SZ)
database_backup="$backup_dir/database-$timestamp.sql.gz"
uploads_backup="$backup_dir/uploads-$timestamp.tar.gz"

mkdir -p "$backup_dir"

PGPASSWORD="$POSTGRES_PASSWORD" pg_dump \
  -h postgres \
  -U "$POSTGRES_USER" \
  "$POSTGRES_DB" | gzip -9 > "$database_backup"

tar -czf "$uploads_backup" -C / uploads

cd "$backup_dir"
sha256sum "$(basename "$database_backup")" "$(basename "$uploads_backup")" \
  > "checksums-$timestamp.txt"

find "$backup_dir" -type f -mtime +14 -delete
echo "Backup completed: $timestamp"
