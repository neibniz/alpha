#!/bin/sh
set -eu

ROOT=${1:-/data/alpha}
case "$ROOT" in
  /data|/data/*) ;;
  *)
    echo "rotation root must be under /data" >&2
    exit 1
    ;;
esac

ENV_FILE="$ROOT/.env"
CONFIG_FILE="$ROOT/config/server/alpha.yaml"
SECRETS_DIR="$ROOT/secrets"

for command in awk chmod cp curl date docker grep jq mkdir mktemp openssl; do
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
backup_dir="$ROOT/backups/security-rotation-$timestamp"
mkdir -p "$backup_dir" "$SECRETS_DIR"
chmod 700 "$backup_dir" "$SECRETS_DIR"
cp -p "$ENV_FILE" "$backup_dir/env"
cp -p "$CONFIG_FILE" "$backup_dir/alpha.yaml"

set -a
# shellcheck disable=SC1090
. "$ENV_FILE"
set +a

for name in ALPHA_POSTGRES_USER ALPHA_POSTGRES_PASSWORD ALPHA_POSTGRES_DB ALPHA_REDIS_PASSWORD ALPHA_CLICKHOUSE_USER ALPHA_CLICKHOUSE_PASSWORD; do
  eval "value=\${$name:-}"
  if [ -z "$value" ]; then
    echo "required environment value is empty: $name" >&2
    exit 1
  fi
done

new_postgres_password=$(openssl rand -hex 32)
new_redis_password=$(openssl rand -hex 32)
new_clickhouse_password=$(openssl rand -hex 32)
new_jwt_secret=$(openssl rand -hex 48)
new_admin_password=$(openssl rand -hex 24)
new_node_master_secret=$(openssl rand -hex 48)
new_node_deploy_secret=$(openssl rand -hex 48)
for value in "$new_postgres_password" "$new_redis_password" "$new_clickhouse_password" "$new_jwt_secret" "$new_admin_password" "$new_node_master_secret" "$new_node_deploy_secret"; do
  case "$value" in
    *[!0-9a-f]*)
      echo "generated credential has an unexpected format" >&2
      exit 1
      ;;
  esac
done
rotation_secrets="$SECRETS_DIR/rotation-$timestamp.env"
cat >"$rotation_secrets" <<EOF
ALPHA_POSTGRES_PASSWORD=$new_postgres_password
ALPHA_REDIS_PASSWORD=$new_redis_password
ALPHA_CLICKHOUSE_PASSWORD=$new_clickhouse_password
ALPHA_JWT_SECRET=$new_jwt_secret
ALPHA_ADMIN_PASSWORD=$new_admin_password
ALPHA_NODE_MASTER_SECRET=$new_node_master_secret
ALPHA_NODE_DEPLOY_SECRET=$new_node_deploy_secret
EOF
chmod 600 "$rotation_secrets"
printf '%s\n' "$new_admin_password" >"$SECRETS_DIR/admin-password"
printf '%s\n' "$new_node_deploy_secret" >"$SECRETS_DIR/node-deploy-token"
chmod 600 "$SECRETS_DIR/admin-password" "$SECRETS_DIR/node-deploy-token"

cd "$ROOT"

docker compose exec -T \
  postgres pg_dump \
  --username "$ALPHA_POSTGRES_USER" \
  --dbname "$ALPHA_POSTGRES_DB" \
  --format custom \
  --file "/tmp/alpha-security-$timestamp.dump"
docker compose cp "postgres:/tmp/alpha-security-$timestamp.dump" "$backup_dir/postgres.dump" >/dev/null
docker compose exec -T postgres rm -f "/tmp/alpha-security-$timestamp.dump"
chmod 600 "$backup_dir/postgres.dump"

admin_count=$(docker compose exec -T \
  postgres psql \
  --username "$ALPHA_POSTGRES_USER" \
  --dbname "$ALPHA_POSTGRES_DB" \
  --tuples-only --no-align \
  --command 'SELECT count(*) FROM "user" WHERE is_admin = TRUE')
admin_count=$(printf '%s' "$admin_count" | tr -d '[:space:]')
if [ "$admin_count" -lt 1 ]; then
  echo "no administrator account was found" >&2
  exit 1
fi

docker compose exec -T \
  postgres psql \
  --username "$ALPHA_POSTGRES_USER" \
  --dbname "$ALPHA_POSTGRES_DB" \
  --set ON_ERROR_STOP=1 \
  --command "CREATE EXTENSION IF NOT EXISTS pgcrypto; UPDATE \"user\" SET password = crypt('$new_admin_password', gen_salt('bf', 12)), algo = 'bcrypt', updated_at = CURRENT_TIMESTAMP WHERE is_admin = TRUE"

docker compose exec -T \
  postgres psql \
  --username "$ALPHA_POSTGRES_USER" \
  --dbname "$ALPHA_POSTGRES_DB" \
  --set ON_ERROR_STOP=1 \
  --command "DELETE FROM system WHERE category = 'server' AND key = 'NodeSecret'"

docker compose exec -T \
  postgres psql \
  --username "$ALPHA_POSTGRES_USER" \
  --dbname "$ALPHA_POSTGRES_DB" \
  --set ON_ERROR_STOP=1 \
  --set "postgres_user=$ALPHA_POSTGRES_USER" \
  --set "new_postgres_password=$new_postgres_password" <<'SQL'
SELECT format('ALTER ROLE %I WITH PASSWORD %L', :'postgres_user', :'new_postgres_password') \gexec
SQL

replace_env_value() {
  key=$1
  value=$2
  input=$3
  output=$(mktemp "$ROOT/.env.rotate.XXXXXX")
  awk -F= -v target="$key" -v replacement="$value" '
    BEGIN { found = 0 }
    $1 == target { print target "=" replacement; found = 1; next }
    { print }
    END { if (!found) print target "=" replacement }
  ' "$input" >"$output"
  chmod 600 "$output"
  mv "$output" "$input"
}

replace_env_value ALPHA_POSTGRES_PASSWORD "$new_postgres_password" "$ENV_FILE"
replace_env_value ALPHA_REDIS_PASSWORD "$new_redis_password" "$ENV_FILE"
replace_env_value ALPHA_CLICKHOUSE_PASSWORD "$new_clickhouse_password" "$ENV_FILE"
export ALPHA_POSTGRES_PASSWORD="$new_postgres_password"
export ALPHA_REDIS_PASSWORD="$new_redis_password"
export ALPHA_CLICKHOUSE_PASSWORD="$new_clickhouse_password"

config_tmp=$(mktemp "$ROOT/alpha.yaml.rotate.XXXXXX")
awk \
  -v jwt="$new_jwt_secret" \
  -v postgres="$new_postgres_password" \
  -v redis="$new_redis_password" \
  -v clickhouse="$new_clickhouse_password" \
  -v admin="$new_admin_password" \
  -v node_master="$new_node_master_secret" \
  -v node_deploy="$new_node_deploy_secret" '
  BEGIN { node_seen = 0; node_master_written = 0; node_deploy_written = 0 }
  /^[^[:space:]]/ {
    if (section == "Node" && !node_master_written) {
      print "  NodeSecret: \"" node_master "\""
      node_master_written = 1
    }
    if (section == "Node" && !node_deploy_written) {
      print "  NodeDeploySecret: \"" node_deploy "\""
      node_deploy_written = 1
    }
    section = $1
    sub(/:$/, "", section)
    if (section == "Node") node_seen = 1
  }
  section == "JwtAuth" && $1 == "AccessSecret:" {
    print "  AccessSecret: \"" jwt "\""
    next
  }
  section == "Postgres" && $1 == "Password:" {
    print "  Password: \"" postgres "\""
    next
  }
  section == "Redis" && $1 == "Pass:" {
    print "  Pass: \"" redis "\""
    next
  }
  section == "ClickHouse" && $1 == "Password:" {
    print "  Password: \"" clickhouse "\""
    next
  }
  section == "Administrator" && $1 == "Password:" {
    print "  Password: \"" admin "\""
    next
  }
  section == "Node" && $1 == "NodeSecret:" {
    print "  NodeSecret: \"" node_master "\""
    node_master_written = 1
    next
  }
  section == "Node" && $1 == "NodeDeploySecret:" {
    print "  NodeDeploySecret: \"" node_deploy "\""
    node_deploy_written = 1
    next
  }
  { print }
  END {
    if (section == "Node" && !node_master_written) {
      print "  NodeSecret: \"" node_master "\""
    }
    if (section == "Node" && !node_deploy_written) {
      print "  NodeDeploySecret: \"" node_deploy "\""
    } else if (!node_seen) {
      print ""
      print "Node:"
      print "  NodeSecret: \"" node_master "\""
      print "  NodeDeploySecret: \"" node_deploy "\""
    }
  }
' "$CONFIG_FILE" >"$config_tmp"
chmod 600 "$config_tmp"
mv "$config_tmp" "$CONFIG_FILE"

chmod 600 "$ENV_FILE" "$CONFIG_FILE" "$SECRETS_DIR/admin-password" "$SECRETS_DIR/node-deploy-token"

"$ROOT/prepare-runtime-permissions.sh" "$ROOT"

docker compose up -d --force-recreate postgres redis clickhouse server

deadline=$(( $(date +%s) + 180 ))
while :; do
  ready=true
  for container in alpha-postgres alpha-redis alpha-clickhouse; do
    status=$(docker inspect --format '{{if .State.Health}}{{.State.Health.Status}}{{else}}{{.State.Status}}{{end}}' "$container")
    if [ "$status" != "healthy" ] && [ "$status" != "running" ]; then
      ready=false
    fi
  done
  server_status=$(docker inspect --format '{{.State.Status}}' alpha-server)
  if [ "$ready" = true ] && [ "$server_status" = "running" ]; then
    break
  fi
  if [ "$(date +%s)" -ge "$deadline" ]; then
    echo "services did not become ready after credential rotation" >&2
    exit 1
  fi
  sleep 3
done

for pattern in 'auth:session_id:*' 'auth:user_sessions:*' 'cache:user:*' 'system:node_config'; do
  docker compose exec -T redis redis-cli \
      --no-auth-warning \
      -a "$new_redis_password" \
      --scan --pattern "$pattern" | while IFS= read -r key; do
    [ -n "$key" ] || continue
    docker compose exec -T redis redis-cli \
      --no-auth-warning \
      -a "$new_redis_password" \
      DEL "$key" >/dev/null
  done
done

app_deadline=$(( $(date +%s) + 90 ))
while :; do
  heartbeat=$(curl --fail --silent --max-time 5 http://127.0.0.1:27001/v1/common/heartbeat 2>/dev/null || true)
  if printf '%s' "$heartbeat" | grep -q '"status":true'; then
    break
  fi
  if [ "$(date +%s)" -ge "$app_deadline" ]; then
    echo "server heartbeat did not become ready after credential rotation" >&2
    exit 1
  fi
  sleep 2
done

admin_email=$(awk '
  /^[^[:space:]]/ { section = $1; sub(/:$/, "", section) }
  section == "Administrator" && $1 == "Email:" {
    value = $2
    gsub(/^"|"$/, "", value)
    print value
    exit
  }
' "$CONFIG_FILE")
login_payload=$(jq -cn \
  --arg email "$admin_email" \
  --arg password "$new_admin_password" \
  '{email:$email,password:$password}')
login_response=$(curl --fail --silent --show-error --max-time 10 \
  -H 'Content-Type: application/json' \
  --data "$login_payload" \
  http://127.0.0.1:27001/v1/auth/login)
printf '%s' "$login_response" | grep -q '"token"'

rm -f "$rotation_secrets"

echo "credential rotation completed"
echo "administrator password file: $SECRETS_DIR/admin-password"
echo "node deployment token file: $SECRETS_DIR/node-deploy-token"
echo "backup directory: $backup_dir"
