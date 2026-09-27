#!/bin/zsh
set -euo pipefail

umask 077
script_dir="${0:A:h}"
project_dir="${script_dir:h}"
credentials_dir="$project_dir/.secrets"
credentials_file="$credentials_dir/admin-login.txt"
docker_bin="/Applications/Docker.app/Contents/Resources/bin/docker"

mkdir -p "$credentials_dir"

if [[ ! -f "$credentials_file" ]]; then
  admin_password=$(openssl rand -hex 16)
  printf 'Computer Shop POS\nUsername: admin\nPassword: %s\n' "$admin_password" > "$credentials_file"
  chmod 600 "$credentials_file"
else
  admin_password=$(sed -n 's/^Password: //p' "$credentials_file")
fi

if [[ -z "$admin_password" ]]; then
  echo "The saved admin credential file is invalid." >&2
  exit 1
fi

admin_hash=$(/usr/sbin/htpasswd -bnBC 12 admin "$admin_password" | cut -d: -f2)

printf "UPDATE users SET password_hash = '%s', is_active = (username = 'admin');\n" "$admin_hash" | \
  "$docker_bin" exec -i pos-postgres sh -c \
    'psql -v ON_ERROR_STOP=1 -U "$POSTGRES_USER" -d "$POSTGRES_DB"' >/dev/null

echo "Admin account secured; demo accounts disabled."
