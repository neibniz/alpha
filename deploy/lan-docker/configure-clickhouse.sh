#!/bin/sh

set -eu
umask 077

deploy_root=${1:-/data/alpha}
case "$deploy_root" in
  /data/*) ;;
  *)
    printf 'deployment root must be below /data: %s\n' "$deploy_root" >&2
    exit 1
    ;;
esac

env_file="$deploy_root/.env"
server_config="$deploy_root/config/server/alpha.yaml"

if [ ! -f "$env_file" ] || [ ! -f "$server_config" ]; then
  printf 'missing deployment files below %s\n' "$deploy_root" >&2
  exit 1
fi

ensure_env() {
  key=$1
  value=$2
  if ! grep -q "^${key}=" "$env_file"; then
    printf '%s=%s\n' "$key" "$value" >>"$env_file"
  fi
}

ensure_env ALPHA_CLICKHOUSE_DB alpha_analytics
ensure_env ALPHA_CLICKHOUSE_USER alpha
ensure_env ALPHA_CLICKHOUSE_PASSWORD "$(openssl rand -hex 24)"
chmod 0600 "$env_file"

clickhouse_db=$(sed -n 's/^ALPHA_CLICKHOUSE_DB=//p' "$env_file" | tail -n 1)
clickhouse_user=$(sed -n 's/^ALPHA_CLICKHOUSE_USER=//p' "$env_file" | tail -n 1)
clickhouse_password=$(sed -n 's/^ALPHA_CLICKHOUSE_PASSWORD=//p' "$env_file" | tail -n 1)
clickhouse_db=$(printf '%s' "$clickhouse_db" | sed 's/\\/\\\\/g; s/"/\\"/g')
clickhouse_user=$(printf '%s' "$clickhouse_user" | sed 's/\\/\\\\/g; s/"/\\"/g')
clickhouse_password=$(printf '%s' "$clickhouse_password" | sed 's/\\/\\\\/g; s/"/\\"/g')

temp_config="$deploy_root/config/server/.alpha.yaml.clickhouse"
awk '
  /^ClickHouse:[[:space:]]*$/ { skipping = 1; next }
  skipping && /^[[:alnum:]_]+:[[:space:]]*/ { skipping = 0 }
  !skipping { print }
' "$server_config" >"$temp_config"

{
  printf '\nClickHouse:\n'
  printf '  Enabled: true\n'
  printf '  Addr: "clickhouse:9000"\n'
  printf '  Database: "%s"\n' "$clickhouse_db"
  printf '  Username: "%s"\n' "$clickhouse_user"
  printf '  Password: "%s"\n' "$clickhouse_password"
  printf '  AccessRetentionDays: 90\n'
  printf '  TrafficRawRetentionHours: 72\n'
  printf '  TrafficRetentionDays: 90\n'
  printf '  TrafficHourlyRetentionDays: 730\n'
} >>"$temp_config"

chmod --reference="$server_config" "$temp_config"
mv "$temp_config" "$server_config"

"$deploy_root/prepare-runtime-permissions.sh" "$deploy_root"
