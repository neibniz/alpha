#!/bin/sh

set -eu

action=${1:-verify}
deploy_root=${2:-/data/alpha}
case "$deploy_root" in
  /data/*) ;;
  *)
    printf 'deployment root must be below /data: %s\n' "$deploy_root" >&2
    exit 1
    ;;
esac

case "$action" in
  setup | check | cleanup | verify) ;;
  *)
    printf 'usage: %s [setup|check|cleanup|verify] [/data/deployment-root]\n' "$0" >&2
    exit 1
    ;;
esac

cd "$deploy_root"
state_file="$deploy_root/data/traffic-series-verification.state"

db_query() {
  docker compose exec -T postgres sh -c \
    'psql -qAt -v ON_ERROR_STOP=1 -U "$POSTGRES_USER" -d "$POSTGRES_DB" -c "$1"' \
    _ "$1"
}

clickhouse_query() {
  docker compose exec -T clickhouse sh -c \
    'clickhouse-client --user "$CLICKHOUSE_USER" --password "$CLICKHOUSE_PASSWORD" --query "$1"' \
    _ "$1"
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

state_value() {
  awk -F= -v key="$1" '$1 == key { print $2; exit }' "$state_file"
}

require_number() {
  case "$2" in
    '' | *[!0-9]*)
      printf 'invalid %s in verification state\n' "$1" >&2
      exit 1
      ;;
  esac
}

cleanup_fixture() {
  if [ ! -f "$state_file" ]; then
    return
  fi

  fixture_subscribe_id=$(state_value fixture_subscribe_id)
  foreign_subscribe_id=$(state_value foreign_subscribe_id)
  product_id=$(state_value product_id)
  require_number fixture_subscribe_id "$fixture_subscribe_id"
  require_number foreign_subscribe_id "$foreign_subscribe_id"
  require_number product_id "$product_id"

  clickhouse_query "ALTER TABLE alpha_analytics.traffic_events DELETE WHERE user_subscribe_id IN ($fixture_subscribe_id, $foreign_subscribe_id) SETTINGS mutations_sync = 1" >/dev/null
  clickhouse_query "ALTER TABLE alpha_analytics.traffic_minute DELETE WHERE user_subscribe_id IN ($fixture_subscribe_id, $foreign_subscribe_id) SETTINGS mutations_sync = 1" >/dev/null
  clickhouse_query "ALTER TABLE alpha_analytics.traffic_hourly DELETE WHERE user_subscribe_id IN ($fixture_subscribe_id, $foreign_subscribe_id) SETTINGS mutations_sync = 1" >/dev/null
  db_query "DELETE FROM user_subscribe WHERE id IN ($fixture_subscribe_id, $foreign_subscribe_id)" >/dev/null
  db_query "DELETE FROM subscribe WHERE id = $product_id" >/dev/null
  rm -f "$state_file"
  printf 'traffic_series_fixture_cleanup=ok\n'
}

setup_fixture() {
  cleanup_fixture

  user_id=$(db_query 'SELECT id FROM "user" WHERE is_admin = true AND deleted_at IS NULL ORDER BY id LIMIT 1')
  if [ -z "$user_id" ]; then
    printf 'an active administrator is required for traffic series verification\n' >&2
    exit 1
  fi
  require_number user_id "$user_id"
  foreign_user_id=$(db_query 'SELECT COALESCE(MAX(id), 0) + 1000 FROM "user"')
  require_number foreign_user_id "$foreign_user_id"

  product_id=$(db_query "INSERT INTO subscribe (name, unit_time, traffic, show, sell, sort, created_at, updated_at, nodes, node_tags, language, user_visible) VALUES ('Traffic Chart Verification', 'Month', 10737418240, false, false, 0, NOW(), NOW(), '', '', 'en-US', true) RETURNING id")
  require_number product_id "$product_id"
  suffix=$(date +%s)-$$
  fixture_subscribe_id=$(db_query "INSERT INTO user_subscribe (user_id, order_id, subscribe_id, start_time, expire_time, traffic, download, upload, token, uuid, status, created_at, updated_at, note, product_name, product_snapshot) VALUES ($user_id, 0, $product_id, NOW(), NOW() + INTERVAL '30 days', 10737418240, 0, 0, 'traffic-chart-token-$suffix', 'traffic-chart-uuid-$suffix', 1, NOW(), NOW(), '', 'Traffic Chart Verification', '{}') RETURNING id")
  foreign_subscribe_id=$(db_query "INSERT INTO user_subscribe (user_id, order_id, subscribe_id, start_time, expire_time, traffic, download, upload, token, uuid, status, created_at, updated_at, note, product_name, product_snapshot) VALUES ($foreign_user_id, 0, $product_id, NOW(), NOW() + INTERVAL '30 days', 10737418240, 0, 0, 'traffic-chart-foreign-token-$suffix', 'traffic-chart-foreign-uuid-$suffix', 1, NOW(), NOW(), '', 'Traffic Chart Foreign Verification', '{}') RETURNING id")
  require_number fixture_subscribe_id "$fixture_subscribe_id"
  require_number foreign_subscribe_id "$foreign_subscribe_id"

  umask 077
  printf 'user_id=%s\nforeign_user_id=%s\nproduct_id=%s\nfixture_subscribe_id=%s\nforeign_subscribe_id=%s\n' \
    "$user_id" "$foreign_user_id" "$product_id" "$fixture_subscribe_id" "$foreign_subscribe_id" >"$state_file"

  clickhouse_query "INSERT INTO alpha_analytics.traffic_events (event_id, event_time, ingested_at, server_id, user_id, user_subscribe_id, download, upload) SELECT generateUUIDv4(), now64(3) - toIntervalMinute(number * 2), now64(3), 0, toUInt64($user_id), toUInt64($fixture_subscribe_id), toUInt64((number + 1) * 1048576), toUInt64((number + 1) * 262144) FROM numbers(12)" >/dev/null
  clickhouse_query "INSERT INTO alpha_analytics.traffic_minute (bucket_time, refreshed_at, server_id, user_id, user_subscribe_id, download, upload) SELECT toStartOfMinute(event_time) AS bucket_time, now64(3), server_id, user_id, user_subscribe_id, sum(download), sum(upload) FROM alpha_analytics.traffic_events FINAL WHERE user_subscribe_id = $fixture_subscribe_id GROUP BY bucket_time, server_id, user_id, user_subscribe_id" >/dev/null
  clickhouse_query "INSERT INTO alpha_analytics.traffic_hourly (bucket_time, refreshed_at, server_id, user_id, user_subscribe_id, download, upload) SELECT toStartOfHour(event_time) AS bucket_time, now64(3), server_id, user_id, user_subscribe_id, sum(download), sum(upload) FROM alpha_analytics.traffic_events FINAL WHERE user_subscribe_id = $fixture_subscribe_id GROUP BY bucket_time, server_id, user_id, user_subscribe_id" >/dev/null
  row_count=$(clickhouse_query "SELECT count() FROM alpha_analytics.traffic_events FINAL WHERE user_subscribe_id = $fixture_subscribe_id")
  if [ "$row_count" != "12" ]; then
    printf 'expected 12 ClickHouse fixture rows, got %s\n' "$row_count" >&2
    exit 1
  fi

  printf 'traffic_series_fixture=ok\n'
  printf 'fixture_user_id=%s\nfixture_subscribe_id=%s\n' "$user_id" "$fixture_subscribe_id"
}

login_token() {
  login_type=$1
  email=$(config_value Administrator Email)
  password=$(config_value Administrator Password)
  if [ -z "$email" ] || [ -z "$password" ]; then
    printf 'administrator login configuration is unavailable\n' >&2
    exit 1
  fi
  payload=$(jq -cn --arg email "$email" --arg password "$password" '{email:$email,password:$password}')
  response=$(curl -fsS -H 'Content-Type: application/json' -H "Login-Type: $login_type" \
    --data "$payload" http://127.0.0.1:27001/v1/auth/login)
  token=$(printf '%s' "$response" | jq -r '.data.token // empty')
  if [ -z "$token" ]; then
    printf '%s login failed with application code %s\n' "$login_type" \
      "$(printf '%s' "$response" | jq -r '.code // "unknown"')" >&2
    exit 1
  fi
  printf '%s' "$token"
}

check_fixture() {
  if [ ! -f "$state_file" ]; then
    printf 'traffic series verification fixture is not set up\n' >&2
    exit 1
  fi

  user_id=$(state_value user_id)
  foreign_user_id=$(state_value foreign_user_id)
  fixture_subscribe_id=$(state_value fixture_subscribe_id)
  foreign_subscribe_id=$(state_value foreign_subscribe_id)
  require_number user_id "$user_id"
  require_number foreign_user_id "$foreign_user_id"
  require_number fixture_subscribe_id "$fixture_subscribe_id"
  require_number foreign_subscribe_id "$foreign_subscribe_id"

  admin_token=$(login_token admin)
  user_token=$(login_token user)
  expected_download=81788928
  expected_upload=20447232

  admin_response=$(curl -fsS -G -H "Authorization: $admin_token" \
    --data-urlencode "user_id=$user_id" \
    --data-urlencode "user_subscribe_id=$fixture_subscribe_id" \
    --data-urlencode 'period=1h' \
    http://127.0.0.1:27001/v1/admin/user/subscribe/traffic_series)
  if ! printf '%s' "$admin_response" | jq -e \
    --argjson download "$expected_download" --argjson upload "$expected_upload" \
    '.code == 200 and .data.interval_seconds == 60 and (.data.list | length) == 60 and .data.download == $download and .data.upload == $upload and ([.data.list[] | select(.download > 0 or .upload > 0)] | length) == 12' >/dev/null; then
    printf 'Admin traffic series aggregation verification failed\n' >&2
    exit 1
  fi
  printf 'admin_traffic_series=ok\n'

  user_response=$(curl -fsS -G -H "Authorization: $user_token" \
    --data-urlencode "user_subscribe_id=$fixture_subscribe_id" \
    --data-urlencode 'period=1h' \
    http://127.0.0.1:27001/v1/public/user/subscribe_traffic)
  if ! printf '%s' "$user_response" | jq -e \
    --argjson download "$expected_download" --argjson upload "$expected_upload" \
    '.code == 200 and .data.interval_seconds == 60 and (.data.list | length) == 60 and .data.download == $download and .data.upload == $upload' >/dev/null; then
    printf 'User traffic series aggregation verification failed\n' >&2
    exit 1
  fi
  printf 'user_traffic_series=ok\n'

  admin_denied=$(curl -fsS -G -H "Authorization: $admin_token" \
    --data-urlencode "user_id=$foreign_user_id" \
    --data-urlencode "user_subscribe_id=$fixture_subscribe_id" \
    --data-urlencode 'period=1h' \
    http://127.0.0.1:27001/v1/admin/user/subscribe/traffic_series)
  if printf '%s' "$admin_denied" | jq -e '.code == 200' >/dev/null; then
    printf 'Admin traffic series accepted a mismatched user ownership request\n' >&2
    exit 1
  fi
  printf 'admin_ownership_check=ok\n'

  user_denied=$(curl -fsS -G -H "Authorization: $user_token" \
    --data-urlencode "user_subscribe_id=$foreign_subscribe_id" \
    --data-urlencode 'period=1h' \
    http://127.0.0.1:27001/v1/public/user/subscribe_traffic)
  if printf '%s' "$user_denied" | jq -e '.code == 200' >/dev/null; then
    printf 'User traffic series accepted another user subscription\n' >&2
    exit 1
  fi
  printf 'user_ownership_check=ok\n'
}

case "$action" in
  setup)
    setup_fixture
    ;;
  check)
    check_fixture
    ;;
  cleanup)
    cleanup_fixture
    ;;
  verify)
    setup_fixture
    trap cleanup_fixture EXIT INT TERM
    check_fixture
    cleanup_fixture
    trap - EXIT INT TERM
    printf 'traffic_series_verification=ok\n'
    ;;
esac
