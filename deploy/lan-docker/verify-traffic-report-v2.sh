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

clickhouse_query() {
  docker compose exec -T clickhouse sh -c \
    'clickhouse-client --user "$CLICKHOUSE_USER" --password "$CLICKHOUSE_PASSWORD" --query "$1"' _ "$1"
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

server_id=
product_id=
subscribe_id=
sample_date=
actual_download=0
actual_upload=0

cleanup() {
  if [ -n "$subscribe_id" ]; then
    clickhouse_query "ALTER TABLE alpha_analytics.traffic_events DELETE WHERE user_subscribe_id = $subscribe_id SETTINGS mutations_sync = 1" >/dev/null 2>&1 || true
    clickhouse_query "ALTER TABLE alpha_analytics.traffic_minute DELETE WHERE user_subscribe_id = $subscribe_id SETTINGS mutations_sync = 1" >/dev/null 2>&1 || true
    clickhouse_query "ALTER TABLE alpha_analytics.traffic_hourly DELETE WHERE user_subscribe_id = $subscribe_id SETTINGS mutations_sync = 1" >/dev/null 2>&1 || true
    db_query "DELETE FROM traffic_event_outbox WHERE user_subscribe_id = $subscribe_id" >/dev/null 2>&1 || true
    db_query "DELETE FROM daily_user_traffic WHERE subscribe_id = $subscribe_id" >/dev/null 2>&1 || true
    db_query "DELETE FROM user_subscribe WHERE id = $subscribe_id" >/dev/null 2>&1 || true
  fi
  if [ -n "$server_id" ]; then
    db_query "DELETE FROM traffic_report_receipt WHERE server_id = $server_id" >/dev/null 2>&1 || true
    db_query "DELETE FROM daily_server_traffic WHERE server_id = $server_id" >/dev/null 2>&1 || true
    db_query "DELETE FROM servers WHERE id = $server_id" >/dev/null 2>&1 || true
  fi
  if [ -n "$sample_date" ] \
    && { [ "$actual_download" -gt 0 ] || [ "$actual_upload" -gt 0 ]; }; then
    actual_total=$((actual_download + actual_upload))
    db_query "UPDATE daily_traffic_stat SET download = GREATEST(download - $actual_download, 0), upload = GREATEST(upload - $actual_upload, 0), total = GREATEST(total - $actual_total, 0) WHERE date = '$sample_date'" >/dev/null 2>&1 || true
  fi
  if [ -n "$product_id" ]; then
    db_query "DELETE FROM subscribe WHERE id = $product_id" >/dev/null 2>&1 || true
  fi
}
trap cleanup EXIT INT TERM

node_secret=$(config_value Node NodeSecret)
admin_email=$(config_value Administrator Email)
admin_password=$(config_value Administrator Password)
if [ -z "$node_secret" ] || [ -z "$admin_email" ] || [ -z "$admin_password" ]; then
	printf 'verification credentials are unavailable\n' >&2
  exit 1
fi

user_id=$(db_query 'SELECT id FROM "user" WHERE is_admin = true AND deleted_at IS NULL ORDER BY id LIMIT 1')
if [ -z "$user_id" ]; then
  printf 'an active administrator is required\n' >&2
  exit 1
fi

server_id=$(db_query "INSERT INTO servers (name, address, sort, created_at, updated_at) VALUES ('traffic-report-v2-verification', '127.0.0.1', 1, NOW(), NOW()) RETURNING id")
product_id=$(db_query "INSERT INTO subscribe (name, unit_time, traffic, show, sell, sort, created_at, updated_at, nodes, node_tags, language, user_visible) VALUES ('Traffic Report V2 Verification', 'Month', 10737418240, false, false, 0, NOW(), NOW(), '', '', 'en-US', true) RETURNING id")
suffix=$(date +%s)-$$
subscribe_id=$(db_query "INSERT INTO user_subscribe (user_id, order_id, subscribe_id, start_time, expire_time, traffic, download, upload, token, uuid, status, created_at, updated_at, note, product_name, product_snapshot) VALUES ($user_id, 0, $product_id, NOW(), NOW() + INTERVAL '1 day', 10737418240, 0, 0, 'traffic-v2-token-$suffix', 'traffic-v2-uuid-$suffix', 1, NOW(), NOW(), '', 'Traffic Report V2 Verification', '{}') RETURNING id")

node_token=$(derive_node_token "$node_secret" "$server_id")
report_a=$(uuidgen | tr '[:upper:]' '[:lower:]')
report_b=$(uuidgen | tr '[:upper:]' '[:lower:]')
now_millis=$(date +%s%3N)
sampled_at=$((now_millis - 10000))
sampled_at_b=$((now_millis - 5000))
sample_date=$(date -u -d "@$((sampled_at / 1000))" +%F)

payload_a=$(jq -cn \
  --arg report_a "$report_a" \
  --argjson sampled_at "$sampled_at" \
  --argjson subscribe_id "$subscribe_id" \
  '{reports:[{report_id:$report_a,sampled_at:$sampled_at,interval_seconds:5,sequence:1,traffic:[{uid:$subscribe_id,download:200000,upload:100000}]}]}')

payload_ab=$(jq -cn \
  --arg report_a "$report_a" \
  --arg report_b "$report_b" \
  --argjson sampled_at "$sampled_at" \
  --argjson sampled_at_b "$sampled_at_b" \
  --argjson subscribe_id "$subscribe_id" \
  '{reports:[
    {report_id:$report_a,sampled_at:$sampled_at,interval_seconds:5,sequence:1,traffic:[{uid:$subscribe_id,download:200000,upload:100000}]},
    {report_id:$report_b,sampled_at:$sampled_at_b,interval_seconds:5,sequence:2,traffic:[{uid:$subscribe_id,download:11000,upload:7000}]}
  ]}')

post_report() {
  curl -fsS \
    -H 'Content-Type: application/json' \
    -H "Authorization: Bearer $node_token" \
    -H "X-Alpha-Server-ID: $server_id" \
    --data "$1" \
    'http://127.0.0.1:27001/v1/server/push?protocol=xray' >/dev/null
}

post_report "$payload_a"
post_report "$payload_ab"

verified=false
for _ in $(seq 1 40); do
  usage=$(db_query "SELECT download || '|' || upload FROM user_subscribe WHERE id = $subscribe_id")
  receipts=$(db_query "SELECT count(*) FROM traffic_report_receipt WHERE server_id = $server_id")
  raw=$(clickhouse_query "SELECT count() || '|' || sum(download) || '|' || sum(upload) FROM alpha_analytics.traffic_events FINAL WHERE user_subscribe_id = $subscribe_id")
  minute=$(clickhouse_query "SELECT sum(download) || '|' || sum(upload) FROM alpha_analytics.traffic_minute FINAL WHERE user_subscribe_id = $subscribe_id")
  hourly=$(clickhouse_query "SELECT sum(download) || '|' || sum(upload) FROM alpha_analytics.traffic_hourly FINAL WHERE user_subscribe_id = $subscribe_id")
  if [ "$usage" = "211000|107000" ] && [ "$receipts" = "2" ] && [ "$raw" = "2|211000|107000" ] && [ "$minute" = "211000|107000" ] && [ "$hourly" = "211000|107000" ]; then
    verified=true
    actual_download=211000
    actual_upload=107000
    break
  fi
  sleep 1
done

if [ "$verified" != "true" ]; then
  printf 'traffic report verification failed: usage=%s receipts=%s raw=%s minute=%s hourly=%s\n' \
    "$usage" "$receipts" "$raw" "$minute" "$hourly" >&2
  exit 1
fi

event_times=$(clickhouse_query "SELECT countIf(toUnixTimestamp64Milli(event_time) IN ($sampled_at, $sampled_at_b)) FROM alpha_analytics.traffic_events FINAL WHERE user_subscribe_id = $subscribe_id")
if [ "$event_times" != "2" ]; then
  printf 'traffic event timestamps do not match agent sample times\n' >&2
  exit 1
fi

login_payload=$(jq -cn --arg email "$admin_email" --arg password "$admin_password" '{email:$email,password:$password}')
admin_token=$(curl -fsS -H 'Content-Type: application/json' -H 'Login-Type: admin' --data "$login_payload" http://127.0.0.1:27001/v1/auth/login | jq -r '.data.token // empty')
user_token=$(curl -fsS -H 'Content-Type: application/json' -H 'Login-Type: user' --data "$login_payload" http://127.0.0.1:27001/v1/auth/login | jq -r '.data.token // empty')
if [ -z "$admin_token" ] || [ -z "$user_token" ]; then
	printf 'traffic series verification login failed\n' >&2
	exit 1
fi

admin_series=$(curl -fsS -G -H "Authorization: $admin_token" \
	--data-urlencode "user_id=$user_id" \
	--data-urlencode "user_subscribe_id=$subscribe_id" \
	--data-urlencode 'period=5m' \
	http://127.0.0.1:27001/v1/admin/user/subscribe/traffic_series)
user_series=$(curl -fsS -G -H "Authorization: $user_token" \
	--data-urlencode "user_subscribe_id=$subscribe_id" \
	--data-urlencode 'period=5m' \
	http://127.0.0.1:27001/v1/public/user/subscribe_traffic)
for response in "$admin_series" "$user_series"; do
	if ! printf '%s' "$response" | jq -e '
		.code == 200 and
		.data.interval_seconds == 5 and
		(.data.list | length) == 60 and
		.data.download == 211000 and
		.data.upload == 107000 and
		([.data.list[] | select(.download > 0 or .upload > 0)] | length) == 2
	' >/dev/null; then
		printf 'five-second traffic series API verification failed: %s\n' "$response" >&2
		exit 1
	fi
done

printf 'traffic_report_idempotency=ok\n'
printf 'traffic_sample_timestamps=ok\n'
printf 'traffic_raw_rollup_consistency=ok\n'
printf 'admin_five_second_series=ok\n'
printf 'user_five_second_series=ok\n'
