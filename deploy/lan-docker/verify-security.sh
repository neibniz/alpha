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

cd "$deploy_root"

for command in awk curl date docker grep jq sed stat; do
  if ! command -v "$command" >/dev/null 2>&1; then
    printf 'required command is missing: %s\n' "$command" >&2
    exit 1
  fi
done

db_query() {
  docker compose exec -T postgres sh -c \
    'psql -qAt -v ON_ERROR_STOP=1 -U "$POSTGRES_USER" -d "$POSTGRES_DB" -c "$1"' _ "$1"
}

config_value() {
  section=$1
  key=$2
  awk -v section="$section" -v key="$key" '
    $0 == section ":" { inside = 1; next }
    inside && /^[[:alnum:]_]+:/ { exit }
    inside && $1 == key ":" {
      sub("^[[:space:]]*" key ":[[:space:]]*", "")
      sub(/[[:space:]]+#.*$/, "")
      gsub(/^"|"$/, "")
      print
      exit
    }
  ' config/server/alpha.yaml
}

server_id_one=
server_id_two=
cleanup() {
  if [ -n "$server_id_one" ]; then
    db_query "DELETE FROM servers WHERE id = $server_id_one" >/dev/null 2>&1 || true
  fi
  if [ -n "$server_id_two" ]; then
    db_query "DELETE FROM servers WHERE id = $server_id_two" >/dev/null 2>&1 || true
  fi
}
trap cleanup EXIT INT TERM

for secret_file in \
  .env \
  config/server/alpha.yaml \
  secrets/admin-password \
  secrets/node-deploy-token; do
  mode=$(stat -c '%a' "$secret_file")
  if [ "$mode" != "600" ]; then
    printf 'insecure file mode %s on %s\n' "$mode" "$secret_file" >&2
    exit 1
  fi
done
printf 'credential_file_modes=ok\n'

legacy_secret_count=$(db_query "SELECT count(*) FROM system WHERE category = 'server' AND key = 'NodeSecret'")
if [ "$legacy_secret_count" != "0" ]; then
  printf 'legacy database node secret still exists\n' >&2
  exit 1
fi
printf 'legacy_node_secret_removed=ok\n'

heartbeat=$(curl -fsS --max-time 5 http://127.0.0.1:27001/v1/common/heartbeat)
if ! printf '%s' "$heartbeat" | jq -e '.code == 200 and .data.status == true' >/dev/null; then
  printf 'server heartbeat failed\n' >&2
  exit 1
fi
for target in 3000/login 3001/; do
  status=$(curl -sS --max-time 5 -o /dev/null -w '%{http_code}' "http://127.0.0.1:$target")
  if [ "$status" != "200" ]; then
    printf 'frontend endpoint %s returned HTTP %s\n' "$target" "$status" >&2
    exit 1
  fi
done
printf 'service_health=ok\n'

allowed_headers=$(curl -sS --max-time 5 -o /dev/null -D - \
  -X OPTIONS \
  -H 'Origin: http://192.168.31.192:3000' \
  -H 'Access-Control-Request-Method: POST' \
  -H 'Access-Control-Request-Headers: content-type,login-type' \
  http://127.0.0.1:27001/v1/auth/login)
if ! printf '%s' "$allowed_headers" | grep -qi '^HTTP/.* 204'; then
  printf 'allowed CORS preflight was rejected\n' >&2
  exit 1
fi
if ! printf '%s' "$allowed_headers" | grep -qi '^Access-Control-Allow-Headers:.*Login-Type'; then
  printf 'allowed CORS headers omit Login-Type\n' >&2
  exit 1
fi
denied_status=$(curl -sS --max-time 5 -o /dev/null -w '%{http_code}' \
  -X OPTIONS \
  -H 'Origin: https://evil.example' \
  -H 'Access-Control-Request-Method: POST' \
  http://127.0.0.1:27001/v1/auth/login)
if [ "$denied_status" != "403" ]; then
  printf 'untrusted CORS origin returned HTTP %s\n' "$denied_status" >&2
  exit 1
fi
printf 'cors_boundaries=ok\n'

legacy_status=$(curl -sS --max-time 5 -o /dev/null -w '%{http_code}' \
  'http://127.0.0.1:27001/v2/server/deploy/servers?secret_key=legacy')
if [ "$legacy_status" != "401" ]; then
  printf 'legacy node query authentication returned HTTP %s\n' "$legacy_status" >&2
  exit 1
fi
deploy_token=$(sed -n '1p' secrets/node-deploy-token)
deploy_status=$(curl -sS --max-time 5 -o /dev/null -w '%{http_code}' \
  -H "Authorization: Bearer $deploy_token" \
  http://127.0.0.1:27001/v2/server/deploy/servers)
if [ "$deploy_status" != "200" ]; then
  printf 'node deployment bearer returned HTTP %s\n' "$deploy_status" >&2
  exit 1
fi
printf 'node_deploy_auth=ok\n'

admin_email=$(config_value Administrator Email)
admin_password=$(config_value Administrator Password)
if [ -z "$admin_email" ] || [ -z "$admin_password" ]; then
  printf 'administrator verification identity is unavailable\n' >&2
  exit 1
fi
login_payload=$(jq -cn --arg email "$admin_email" --arg password "$admin_password" '{email:$email,password:$password}')
login_response=$(curl -fsS --max-time 10 \
  -H 'Content-Type: application/json' \
  -H 'Login-Type: admin' \
  --data "$login_payload" \
  http://127.0.0.1:27001/v1/auth/login)
admin_token=$(printf '%s' "$login_response" | jq -r '.data.token // empty')
if [ -z "$admin_token" ]; then
  printf 'administrator login failed\n' >&2
  exit 1
fi
printf 'admin_auth=ok\n'

probe_suffix=$(date +%s)
probe_email="security-log-$probe_suffix@example.invalid"
probe_password="LogProbe-$probe_suffix-DoNotPersist"
probe_payload=$(jq -cn --arg email "$probe_email" --arg password "$probe_password" '{email:$email,password:$password}')
curl -sS --max-time 10 -o /dev/null \
  -H 'Content-Type: application/json' \
  -H 'Login-Type: user' \
  --data "$probe_payload" \
  http://127.0.0.1:27001/v1/auth/login
recent_logs=$(docker logs --since 30s alpha-server 2>&1)
if printf '%s' "$recent_logs" | grep -Fq -- "$probe_email"; then
  printf 'server logs exposed a login identifier\n' >&2
  exit 1
fi
if printf '%s' "$recent_logs" | grep -Fq -- "$probe_password"; then
  printf 'server logs exposed a login password\n' >&2
  exit 1
fi
printf 'runtime_log_redaction=ok\n'

node_config=$(curl -fsS --max-time 5 \
  -H "Authorization: $admin_token" \
  http://127.0.0.1:27001/v1/admin/system/node_config)
if ! printf '%s' "$node_config" | jq -e '.code == 200 and (.data.node_pull_interval | type == "number")' >/dev/null; then
  printf 'administrator node configuration query failed\n' >&2
  exit 1
fi
if printf '%s' "$node_config" | grep -Eqi '"node(_deploy)?_secret"'; then
  printf 'administrator node configuration exposed a master secret\n' >&2
  exit 1
fi
printf 'node_master_not_exposed=ok\n'

server_id_one=$(db_query "INSERT INTO servers (name, address, sort, created_at, updated_at) VALUES ('security-verification-one', '127.0.0.1', 1, NOW(), NOW()) RETURNING id")
server_id_two=$(db_query "INSERT INTO servers (name, address, sort, created_at, updated_at) VALUES ('security-verification-two', '127.0.0.1', 2, NOW(), NOW()) RETURNING id")
node_token_response=$(curl -fsS --max-time 5 \
  -H "Authorization: $admin_token" \
  "http://127.0.0.1:27001/v1/admin/server/node-token?id=$server_id_one")
node_token=$(printf '%s' "$node_token_response" | jq -r '.data.token // empty')
case "$node_token" in
  alpha-node-v1_*) ;;
  *)
    printf 'per-server node token query failed\n' >&2
    exit 1
    ;;
esac

bound_status=$(curl -sS --max-time 5 -o /dev/null -w '%{http_code}' \
  -H "Authorization: Bearer $node_token" \
  -H "X-Alpha-Server-ID: $server_id_one" \
  "http://127.0.0.1:27001/v2/server/$server_id_one/xray-config")
if [ "$bound_status" != "200" ]; then
  printf 'bound node token returned HTTP %s\n' "$bound_status" >&2
  exit 1
fi
cross_server_status=$(curl -sS --max-time 5 -o /dev/null -w '%{http_code}' \
  -H "Authorization: Bearer $node_token" \
  -H "X-Alpha-Server-ID: $server_id_two" \
  "http://127.0.0.1:27001/v2/server/$server_id_two/xray-config")
if [ "$cross_server_status" != "401" ]; then
  printf 'cross-server node token returned HTTP %s\n' "$cross_server_status" >&2
  exit 1
fi
printf 'per_server_node_auth=ok\n'

cleanup
server_id_one=
server_id_two=
trap - EXIT INT TERM
printf 'security_verification_cleanup=ok\n'
