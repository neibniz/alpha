#!/bin/sh

set -eu

deploy_root=${1:-/data/alpha}
server_uid=${ALPHA_SERVER_UID:-65532}
server_gid=${ALPHA_SERVER_GID:-65532}

case "$deploy_root" in
  /data|/data/*) ;;
  *)
    printf 'deployment root must be below /data: %s\n' "$deploy_root" >&2
    exit 1
    ;;
esac
for identifier in "$server_uid" "$server_gid"; do
  case "$identifier" in
    ""|*[!0-9]*)
      printf 'server uid and gid must be numeric\n' >&2
      exit 1
      ;;
  esac
done

config_file="$deploy_root/config/server/alpha.yaml"
keyring_file="$deploy_root/secrets/privacy-keyring.json"
log_dir="$deploy_root/data/server/log"

for file in "$config_file" "$keyring_file"; do
  if [ ! -f "$file" ]; then
    printf 'required server file is missing: %s\n' "$file" >&2
    exit 1
  fi
done

mkdir -p "$log_dir"
chmod 600 "$config_file" "$keyring_file"
chown "$server_uid:$server_gid" "$config_file" "$keyring_file"
chown -R "$server_uid:$server_gid" "$log_dir"
chmod 750 "$log_dir"

printf 'server_runtime_permissions=ok uid=%s gid=%s\n' "$server_uid" "$server_gid"
