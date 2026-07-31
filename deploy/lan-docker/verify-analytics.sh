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

db_query() {
  docker compose exec -T postgres sh -c 'psql -qAt -v ON_ERROR_STOP=1 -U "$POSTGRES_USER" -d "$POSTGRES_DB" -c "$1"' _ "$1"
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

derive_node_token() {
  master=$1
  id=$2
  signature=$(printf 'alpha-node-auth:v1:%s' "$id" | openssl dgst -sha256 -mac HMAC -macopt "key:$master" -binary | openssl base64 -A | tr '+/' '-_' | tr -d '=')
  printf 'alpha-node-v1_%s' "$signature"
}

node_secret=$(config_value Node NodeSecret)
server_id=$(db_query "SELECT id FROM servers ORDER BY id LIMIT 1")
subscribe_id=$(db_query "SELECT id FROM user_subscribe ORDER BY (status = 1) DESC, id LIMIT 1")
admin_email=$(config_value Administrator Email)
admin_password=$(config_value Administrator Password)

if [ -z "$node_secret" ] || [ -z "$admin_email" ] || [ -z "$admin_password" ]; then
  printf 'required verification identity is unavailable\n' >&2
  exit 1
fi

created_server_id=
event_id=
traffic_event_id=
cleanup() {
  if [ -n "$event_id" ]; then
    docker compose exec -T clickhouse sh -c \
      'clickhouse-client --user "$CLICKHOUSE_USER" --password "$CLICKHOUSE_PASSWORD" --query "$1"' \
      _ "ALTER TABLE alpha_analytics.xray_access_events DELETE WHERE event_id = toUUID('$event_id') SETTINGS mutations_sync = 1" \
      >/dev/null 2>&1 || true
  fi
  if [ -n "$traffic_event_id" ]; then
    db_query "DELETE FROM traffic_event_outbox WHERE event_id = '$traffic_event_id'" >/dev/null 2>&1 || true
    docker compose exec -T clickhouse sh -c \
      'clickhouse-client --user "$CLICKHOUSE_USER" --password "$CLICKHOUSE_PASSWORD" --query "$1"' \
      _ "ALTER TABLE alpha_analytics.traffic_events DELETE WHERE event_id = toUUID('$traffic_event_id') SETTINGS mutations_sync = 1" \
      >/dev/null 2>&1 || true
  fi
  if [ -n "$created_server_id" ]; then
    db_query "DELETE FROM servers WHERE id = $created_server_id" >/dev/null 2>&1 || true
  fi
}
trap cleanup EXIT INT TERM

if [ -z "$server_id" ]; then
  server_id=$(db_query "INSERT INTO servers (name, address, sort, created_at, updated_at) VALUES ('analytics-verification', '127.0.0.1', 1, NOW(), NOW()) RETURNING id")
  created_server_id=$server_id
fi

event_id=$(uuidgen | tr '[:upper:]' '[:lower:]')
timestamp=$(date +%s%3N)
node_token=$(derive_node_token "$node_secret" "$server_id")

status=accepted
reason=
if [ -z "$subscribe_id" ]; then
  status=rejected
  reason=analytics-verification
fi

payload=$(jq -cn \
  --arg event_id "$event_id" \
  --argjson timestamp "$timestamp" \
  --argjson subscribe_id "${subscribe_id:-0}" \
  --arg status "$status" \
  --arg reason "$reason" \
  '{events:[
    {event_id:$event_id,timestamp:$timestamp,user_subscribe_id:$subscribe_id,source_ip:"203.0.113.42",source_port:54321,destination_host:"private.example.com",destination_port:443,network:"tcp",status:$status,outbound_tag:"proxy",reason:$reason,parser_version:1},
    {event_id:"invalid-verification-event",timestamp:$timestamp,status:"rejected",parser_version:1}
  ]}')

ingest_response=$(printf '%s' "$payload" | gzip -c | curl -fsS \
  -H 'Content-Type: application/json' \
  -H 'Content-Encoding: gzip' \
  -H "Authorization: Bearer $node_token" \
  -H "X-Alpha-Server-ID: $server_id" \
  --data-binary @- \
  "http://127.0.0.1:27001/v1/server/access-logs")
if ! printf '%s' "$ingest_response" | jq -e '.code == 200 and .data.accepted == 1 and .data.rejected == 1' >/dev/null; then
  printf 'access ingestion failed with application code %s\n' "$(printf '%s' "$ingest_response" | jq -r '.code // "unknown"')" >&2
  exit 1
fi
printf 'access_ingest_gzip_partial_accept=ok\n'

row_count=$(docker compose exec -T clickhouse sh -c \
  'clickhouse-client --user "$CLICKHOUSE_USER" --password "$CLICKHOUSE_PASSWORD" --query "$1"' \
  _ "SELECT count() FROM alpha_analytics.xray_access_events FINAL WHERE event_id = toUUID('$event_id')")
if [ "$row_count" != "1" ]; then
  printf 'unexpected ClickHouse row count after ingestion: %s\n' "$row_count" >&2
  exit 1
fi

privacy_row_count=$(docker compose exec -T clickhouse sh -c \
  'clickhouse-client --user "$CLICKHOUSE_USER" --password "$CLICKHOUSE_PASSWORD" --query "$1"' \
  _ "SELECT countIf(privacy_version = 1 AND startsWith(source_ip, 'enc:v1:') AND source_ip_bidx != '' AND source_ip_display = '203.0.113.*' AND startsWith(destination_host, 'enc:v1:') AND destination_host_bidx != '' AND destination_host_display != 'private.example.com') FROM alpha_analytics.xray_access_events FINAL WHERE event_id = toUUID('$event_id')")
if [ "$privacy_row_count" != "1" ]; then
  printf 'ClickHouse access privacy projection is incomplete\n' >&2
  exit 1
fi
printf 'clickhouse_encryption=ok\n'

login_payload=$(jq -cn --arg email "$admin_email" --arg password "$admin_password" '{email:$email,password:$password}')
login_response=$(curl -fsS \
  -H 'Content-Type: application/json' \
  -H 'Login-Type: admin' \
  --data "$login_payload" \
  http://127.0.0.1:27001/v1/auth/login)
token=$(printf '%s' "$login_response" | jq -r '.data.token // empty')
if [ -z "$token" ]; then
  printf 'admin login failed with application code %s\n' "$(printf '%s' "$login_response" | jq -r '.code // "unknown"')" >&2
  exit 1
fi

current_user_response=$(curl -fsS \
  -H "Authorization: $token" \
  http://127.0.0.1:27001/v1/admin/user/current)
if printf '%s' "$current_user_response" | grep -F "$admin_email" >/dev/null; then
  printf 'current user response exposed the plaintext admin email\n' >&2
  exit 1
fi
printf 'identity_masking=ok\n'

query_response=$(curl -fsS \
  -H "Authorization: $token" \
  "http://127.0.0.1:27001/v1/admin/log/proxy/access?size=200&start_time=$((timestamp - 60000))&end_time=$((timestamp + 60000))")
if ! printf '%s' "$query_response" | jq -e --arg event_id "$event_id" 'any(.data.list[]; .event_id == $event_id)' >/dev/null; then
  printf 'admin access query did not return the inserted event (code=%s, rows=%s, message=%s)\n' \
    "$(printf '%s' "$query_response" | jq -r '.code // "unknown"')" \
    "$(printf '%s' "$query_response" | jq -r '(.data.list // []) | length')" \
    "$(printf '%s' "$query_response" | jq -r '.msg // ""')" >&2
  exit 1
fi
printf 'admin_query=ok\n'

filtered_query_response=$(curl -fsS \
  -H "Authorization: $token" \
  "http://127.0.0.1:27001/v1/admin/log/proxy/access?size=200&start_time=$((timestamp - 60000))&end_time=$((timestamp + 60000))&source_ip=203.0.113.42&destination=private.example.com")
if ! printf '%s' "$filtered_query_response" | jq -e --arg event_id "$event_id" 'any(.data.list[]; .event_id == $event_id)' >/dev/null; then
  printf 'blind-index access filters did not return the inserted event\n' >&2
  exit 1
fi
printf 'blind_index_filters=ok\n'

wide_query_response=$(curl -fsS \
  -H "Authorization: $token" \
  "http://127.0.0.1:27001/v1/admin/log/proxy/access?size=200&start_time=$((timestamp - 86400000))&end_time=$((timestamp + 60000))")
if ! printf '%s' "$wide_query_response" | jq -e --arg event_id "$event_id" 'any(.data.list[]; .event_id == $event_id)' >/dev/null; then
  printf 'explicit 24-hour Admin access range did not return the inserted event\n' >&2
  exit 1
fi
printf 'admin_wide_range=ok\n'

default_query_response=$(curl -fsS \
  -H "Authorization: $token" \
  'http://127.0.0.1:27001/v1/admin/log/proxy/access?size=200')
if ! printf '%s' "$default_query_response" | jq -e --arg event_id "$event_id" 'any(.data.list[]; .event_id == $event_id)' >/dev/null; then
  clickhouse_times=$(docker compose exec -T clickhouse sh -c \
    'clickhouse-client --user "$CLICKHOUSE_USER" --password "$CLICKHOUSE_PASSWORD" --query "$1"' \
    _ "SELECT toUnixTimestamp64Milli(event_time), toUnixTimestamp64Milli(now64(3)) FROM alpha_analytics.xray_access_events FINAL WHERE event_id = toUUID('$event_id')")
  printf 'default Admin access time range did not return the inserted event (agent=%s, host=%s, clickhouse=%s)\n' \
    "$timestamp" "$(date +%s%3N)" "$clickhouse_times" >&2
  exit 1
fi
printf 'admin_default_range=ok\n'

if ! printf '%s' "$query_response" | jq -e --arg event_id "$event_id" '
  any(.data.list[]; .event_id == $event_id and .source_ip == "203.0.113.*" and .destination != "private.example.com")
' >/dev/null; then
  printf 'admin query returned an unmasked network identifier\n' >&2
  exit 1
fi
printf 'masking=ok\n'

printf 'clickhouse_rows=1\n'

traffic_event_id=$(uuidgen | tr '[:upper:]' '[:lower:]')
db_query "INSERT INTO traffic_event_outbox (event_id, server_id, user_id, user_subscribe_id, download, upload, event_time, created_at) VALUES ('$traffic_event_id', $server_id, 0, 0, 123, 45, NOW(), NOW())" >/dev/null
traffic_synced=false
for _ in $(seq 1 35); do
  outbox_count=$(db_query "SELECT count(*) FROM traffic_event_outbox WHERE event_id = '$traffic_event_id'")
  traffic_count=$(docker compose exec -T clickhouse sh -c \
    'clickhouse-client --user "$CLICKHOUSE_USER" --password "$CLICKHOUSE_PASSWORD" --query "$1"' \
    _ "SELECT count() FROM alpha_analytics.traffic_events FINAL WHERE event_id = toUUID('$traffic_event_id')")
  if [ "$outbox_count" = "0" ] && [ "$traffic_count" = "1" ]; then
    traffic_synced=true
    break
  fi
  sleep 2
done
if [ "$traffic_synced" != "true" ]; then
  printf 'traffic outbox did not reach ClickHouse within the verification window\n' >&2
  exit 1
fi
printf 'traffic_outbox_sync=ok\n'

cleanup
event_id=
traffic_event_id=
created_server_id=
printf 'test_event_cleanup=ok\n'
