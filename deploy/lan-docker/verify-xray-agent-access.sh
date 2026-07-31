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

container_name=alpha-xray-agent-verification
server_id=
verify_root="$deploy_root/data/xray-agent-verification"
access_log="$verify_root/xray-access.log"

cleanup() {
  docker stop "$container_name" >/dev/null 2>&1 || true
  if [ -n "$server_id" ]; then
    clickhouse_query "ALTER TABLE alpha_analytics.xray_access_events DELETE WHERE server_id = $server_id SETTINGS mutations_sync = 1" >/dev/null 2>&1 || true
    db_query "DELETE FROM servers WHERE id = $server_id" >/dev/null 2>&1 || true
  fi
  rm -f "$access_log" "$verify_root/access_spool.jsonl" "$verify_root/access_state.json"
}
trap cleanup EXIT INT TERM

node_secret=$(config_value Node NodeSecret)
if [ -z "$node_secret" ]; then
  printf 'node secret is unavailable\n' >&2
  exit 1
fi

server_id=$(db_query "INSERT INTO servers (name, address, sort, created_at, updated_at) VALUES ('xray-agent-verification', '127.0.0.1', 1, NOW(), NOW()) RETURNING id")
node_token=$(derive_node_token "$node_secret" "$server_id")
mkdir -p "$verify_root"
: >"$access_log"

docker run -d --rm \
  --name "$container_name" \
  --network host \
  -e TZ=Asia/Shanghai \
  -e ALPHA_SERVER_URL=http://127.0.0.1:27001 \
  -e ALPHA_SERVER_ID="$server_id" \
  -e ALPHA_NODE_TOKEN="$node_token" \
  -e XRAY_AGENT_WORK_DIR=/var/lib/alpha/xray-agent \
  -e XRAY_API_PORT=19085 \
  -e XRAY_ACCESS_LOG_PATH=/var/log/alpha/xray-agent/xray-access.log \
  -e XRAY_ACCESS_BATCH_SIZE=100 \
  -e XRAY_ACCESS_FLUSH_INTERVAL=5s \
  -v "$verify_root:/var/lib/alpha/xray-agent" \
  -v "$verify_root:/var/log/alpha/xray-agent" \
  alpha-xray-agent:lan -config "" >/dev/null

i=1
while [ "$i" -le 100 ]; do
  printf '%s from tcp:203.0.113.42:%s rejected  xray-agent-verification-%s\n' \
    "$(date '+%Y/%m/%d %H:%M:%S')" "$((20000 + i))" "$i" >>"$access_log"
  i=$((i + 1))
done

stored=0
for _ in $(seq 1 30); do
  stored=$(clickhouse_query "SELECT count() FROM alpha_analytics.xray_access_events FINAL WHERE server_id = $server_id")
  if [ "$stored" = "100" ]; then
    break
  fi
  sleep 1
done
if [ "$stored" != "100" ]; then
  docker logs --tail=100 "$container_name" >&2 || true
  printf 'xray-agent access batch stored %s of 100 rows\n' "$stored" >&2
  exit 1
fi

pending=$(jq -r '.pending // -1' "$verify_root/access_state.json")
if [ "$pending" != "0" ]; then
  printf 'xray-agent spool still has %s pending rows\n' "$pending" >&2
  exit 1
fi

printf 'xray_agent_access_batch=ok\n'
printf 'xray_agent_spool_ack=ok\n'

cleanup
server_id=
trap - EXIT INT TERM
printf 'xray_agent_test_cleanup=ok\n'
