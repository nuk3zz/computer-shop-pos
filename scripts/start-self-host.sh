#!/bin/zsh
set -euo pipefail

script_dir="${0:A:h}"
project_dir="${script_dir:h}"
docker_bin="/usr/local/bin/docker"
export PATH="/Applications/Docker.app/Contents/Resources/bin:$PATH"

mkdir -p "$project_dir/uploads" "$project_dir/runtime-logs" "$project_dir/backups/self-host"

if [[ ! -x "$docker_bin" ]]; then
  docker_bin="/Applications/Docker.app/Contents/Resources/bin/docker"
fi

if ! "$docker_bin" info >/dev/null 2>&1; then
  open -gja Docker
  for attempt in {1..90}; do
    if "$docker_bin" info >/dev/null 2>&1; then
      break
    fi
    sleep 2
  done
fi

if ! "$docker_bin" info >/dev/null 2>&1; then
  echo "Docker Desktop did not become ready." >&2
  exit 1
fi

cd "$project_dir"
"$docker_bin" compose --env-file .env -f docker-compose.yml up -d
"$project_dir/scripts/secure-first-admin.sh"
