#!/bin/sh
set -eu

ROOT=${1:-/data/alpha}
LAN_HOST=${2:-192.168.31.192}

case "$ROOT" in
  /data|/data/*) ;;
  *)
    echo "security configuration root must be under /data" >&2
    exit 1
    ;;
esac

case "$LAN_HOST" in
  *[!0-9A-Za-z.:-]*)
    echo "LAN host contains unsupported characters" >&2
    exit 1
    ;;
esac

ENV_FILE="$ROOT/.env"
CONFIG_FILE="$ROOT/config/server/alpha.yaml"
SECRETS_DIR="$ROOT/secrets"

for command in awk chmod cp curl date docker grep mkdir mktemp openssl; do
  if ! command -v "$command" >/dev/null 2>&1; then
    echo "required command is missing: $command" >&2
    exit 1
  fi
done

if [ ! -f "$ENV_FILE" ] || [ ! -f "$CONFIG_FILE" ] || [ ! -f "$ROOT/docker-compose.yml" ]; then
  echo "alpha deployment files are incomplete under $ROOT" >&2
  exit 1
fi

umask 077
timestamp=$(date -u +%Y%m%dT%H%M%SZ)
backup_dir="$ROOT/backups/runtime-security-$timestamp"
mkdir -p "$backup_dir" "$SECRETS_DIR"
chmod 700 "$backup_dir" "$SECRETS_DIR"
cp -p "$CONFIG_FILE" "$backup_dir/alpha.yaml"

set -a
# shellcheck disable=SC1090
. "$ENV_FILE"
set +a

for name in ALPHA_POSTGRES_USER ALPHA_POSTGRES_DB ALPHA_REDIS_PASSWORD; do
  eval "value=\${$name:-}"
  if [ -z "$value" ]; then
    echo "required environment value is empty: $name" >&2
    exit 1
  fi
done

node_master_secret=$(openssl rand -hex 48)
node_deploy_secret=$(openssl rand -hex 48)

docker compose -f "$ROOT/docker-compose.yml" exec -T \
  postgres psql \
  --username "$ALPHA_POSTGRES_USER" \
  --dbname "$ALPHA_POSTGRES_DB" \
  --set ON_ERROR_STOP=1 \
  --command "DELETE FROM system WHERE category = 'server' AND key = 'NodeSecret'"

config_tmp=$(mktemp "$ROOT/alpha.yaml.security.XXXXXX")
awk \
  -v node_master="$node_master_secret" \
  -v node_deploy="$node_deploy_secret" \
  -v user_origin="http://$LAN_HOST:3000" \
  -v admin_origin="http://$LAN_HOST:3001" '
  /^(HTTP|Node):[[:space:]]*$/ {
    skipping = 1
    next
  }
  skipping && /^[^[:space:]]/ {
    skipping = 0
  }
  !skipping { print }
  END {
    print ""
    print "HTTP:"
    print "  AllowedOrigins:"
    print "    - \"" user_origin "\""
    print "    - \"" admin_origin "\""
    print "  TrustedProxies:"
    print "    - \"127.0.0.1\""
    print "    - \"::1\""
    print "  MaxBodyBytes: 8388608"
    print ""
    print "Node:"
    print "  NodeSecret: \"" node_master "\""
    print "  NodeDeploySecret: \"" node_deploy "\""
  }
' "$CONFIG_FILE" >"$config_tmp"
chmod 600 "$config_tmp"
mv "$config_tmp" "$CONFIG_FILE"

printf '%s\n' "$node_deploy_secret" >"$SECRETS_DIR/node-deploy-token"
chmod 600 "$CONFIG_FILE" "$SECRETS_DIR/node-deploy-token"

"$ROOT/prepare-runtime-permissions.sh" "$ROOT"

docker compose -f "$ROOT/docker-compose.yml" exec -T redis redis-cli \
  --no-auth-warning \
  -a "$ALPHA_REDIS_PASSWORD" \
  DEL system:node_config >/dev/null

docker compose -f "$ROOT/docker-compose.yml" up -d --force-recreate server

deadline=$(( $(date +%s) + 120 ))
while :; do
  heartbeat=$(curl --fail --silent --max-time 5 http://127.0.0.1:27001/v1/common/heartbeat 2>/dev/null || true)
  if printf '%s' "$heartbeat" | grep -q '"status":true'; then
    break
  fi
  if [ "$(date +%s)" -ge "$deadline" ]; then
    echo "server heartbeat did not become ready after security configuration" >&2
    exit 1
  fi
  sleep 2
done

legacy_status=$(curl --silent --output /dev/null --write-out '%{http_code}' \
  --max-time 5 \
  'http://127.0.0.1:27001/v2/server/deploy/servers?secret_key=legacy')
if [ "$legacy_status" != "401" ]; then
  echo "legacy node query authentication was not rejected (HTTP $legacy_status)" >&2
  exit 1
fi

deploy_status=$(curl --silent --output /dev/null --write-out '%{http_code}' \
  --max-time 5 \
  -H "Authorization: Bearer $node_deploy_secret" \
  http://127.0.0.1:27001/v2/server/deploy/servers)
if [ "$deploy_status" != "200" ]; then
  echo "node deployment authentication failed (HTTP $deploy_status)" >&2
  exit 1
fi

echo "runtime security configuration completed"
echo "node deployment token file: $SECRETS_DIR/node-deploy-token"
echo "backup directory: $backup_dir"
