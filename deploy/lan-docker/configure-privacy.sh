#!/bin/sh

set -eu

deploy_root=${1:-/data/alpha}
case "$deploy_root" in
  /data/*) ;;
  *)
    printf 'deployment root must be below /data: %s\n' "$deploy_root" >&2
    exit 1
    ;;
esac

server_config="$deploy_root/config/server/alpha.yaml"
secret_dir="$deploy_root/secrets"
keyring="$secret_dir/privacy-keyring.json"

if [ ! -f "$server_config" ]; then
  printf 'missing server config: %s\n' "$server_config" >&2
  exit 1
fi

mkdir -p "$secret_dir"
chmod 0700 "$secret_dir"
if [ ! -f "$keyring" ]; then
  umask 077
  encryption_key=$(openssl rand -base64 32 | tr -d '\n')
  index_key=$(openssl rand -base64 32 | tr -d '\n')
  temp_keyring="$secret_dir/.privacy-keyring.json.tmp"
  printf '{\n  "active_key_id": "k1",\n  "encryption_keys": {"k1": "%s"},\n  "index_key": "%s"\n}\n' \
    "$encryption_key" "$index_key" >"$temp_keyring"
  chmod 0600 "$temp_keyring"
  mv "$temp_keyring" "$keyring"
fi
chmod 0600 "$keyring"

temp_config="$deploy_root/config/server/.alpha.yaml.privacy"
awk '
  /^Privacy:[[:space:]]*$/ { skipping = 1; next }
  skipping && /^[[:alnum:]_]+:[[:space:]]*/ { skipping = 0 }
  !skipping { print }
' "$server_config" >"$temp_config"

{
  printf '\nPrivacy:\n'
  printf '  Enabled: true\n'
  printf '  KeyFile: /run/secrets/alpha_privacy_keyring\n'
} >>"$temp_config"

chmod --reference="$server_config" "$temp_config"
mv "$temp_config" "$server_config"

"$deploy_root/prepare-runtime-permissions.sh" "$deploy_root"

printf 'privacy encryption configured below %s\n' "$deploy_root"
