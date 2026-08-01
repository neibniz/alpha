#!/bin/sh

set -eu

ROOT=${1:-/data/alpha}
STAGING=${2:-}
RELEASE_ID=${3:-}
LAN_HOST=${4:-}
SUBSCRIBE_PATH=${5:-/api/subscribe}

case "$ROOT" in
  /data|/data/*) ;;
  *)
    echo "deployment root must be under /data" >&2
    exit 1
    ;;
esac
case "$STAGING" in
  "$ROOT"/.staging/*) ;;
  *)
    echo "staging directory must be below $ROOT/.staging" >&2
    exit 1
    ;;
esac
case "$RELEASE_ID" in
  ""|*[!0-9A-Za-z._-]*)
    echo "release id contains unsupported characters" >&2
    exit 1
    ;;
esac
case "$LAN_HOST" in
  ""|*[!0-9A-Za-z.:-]*)
    echo "LAN host contains unsupported characters" >&2
    exit 1
    ;;
esac
case "$SUBSCRIBE_PATH" in
  /*) ;;
  *)
    echo "subscription path must be an absolute URL path" >&2
    exit 1
    ;;
esac
case "$SUBSCRIBE_PATH" in
  *[\?\#\ ]*)
    echo "subscription path must not contain a query, fragment, or whitespace" >&2
    exit 1
    ;;
esac

for command in awk cp curl date docker grep mkdir mktemp mv rm sha256sum sleep tar; do
  if ! command -v "$command" >/dev/null 2>&1; then
    echo "required command is missing: $command" >&2
    exit 1
  fi
done

FRONTEND_STAGE="$STAGING/frontend"
DEPLOY_STAGE="$STAGING/deploy"
SERVER_STAGE="$STAGING/src/server"
ADMIN_RELEASE="$ROOT/releases/admin/$RELEASE_ID"
USER_RELEASE="$ROOT/releases/user-next/$RELEASE_ID"
BACKUP_DIR="$ROOT/backups/release-$RELEASE_ID"
STATE_DIR="$ROOT/.deploy-state"
COMPOSE_FILE="$ROOT/docker-compose.yml"
USER_PREFLIGHT_CONTAINER="alpha-user-preflight-$RELEASE_ID"
ADMIN_PREFLIGHT_CONTAINER="alpha-admin-preflight-$RELEASE_ID"

for path in \
  "$FRONTEND_STAGE/alpha-admin-web.tar.gz" \
  "$FRONTEND_STAGE/alpha-admin-web.tar.gz.sha256" \
  "$FRONTEND_STAGE/alpha-user-next.tar.gz" \
  "$FRONTEND_STAGE/alpha-user-next.tar.gz.sha256" \
  "$DEPLOY_STAGE/docker-compose.yml" \
  "$DEPLOY_STAGE/docker/frontend.dist.Dockerfile" \
  "$DEPLOY_STAGE/docker/frontend.user.Dockerfile" \
  "$DEPLOY_STAGE/docker/nginx/frontend.conf" \
  "$DEPLOY_STAGE/prepare-runtime-permissions.sh" \
  "$SERVER_STAGE/Dockerfile"; do
  if [ ! -f "$path" ]; then
    echo "release input is missing: $path" >&2
    exit 1
  fi
done

(cd "$FRONTEND_STAGE" && sha256sum -c alpha-admin-web.tar.gz.sha256)
(cd "$FRONTEND_STAGE" && sha256sum -c alpha-user-next.tar.gz.sha256)

if [ -e "$ADMIN_RELEASE" ] || [ -e "$USER_RELEASE" ]; then
  echo "release already exists: $RELEASE_ID" >&2
  exit 1
fi

cleanup_candidate=true
cleanup_candidate_release() {
  status=$?
  trap - EXIT INT TERM
  if [ "$cleanup_candidate" = true ]; then
    echo "candidate build failed; removing incomplete release artifacts" >&2
    docker rm -f "$USER_PREFLIGHT_CONTAINER" "$ADMIN_PREFLIGHT_CONTAINER" >/dev/null 2>&1 || true
    rm -rf "$ADMIN_RELEASE" "$USER_RELEASE"
    docker image rm "alpha-server:candidate-$RELEASE_ID" >/dev/null 2>&1 || true
    docker image rm "alpha-frontend-user:candidate-$RELEASE_ID" >/dev/null 2>&1 || true
    docker image rm "alpha-frontend-admin:candidate-$RELEASE_ID" >/dev/null 2>&1 || true
  fi
  exit "$status"
}
trap cleanup_candidate_release EXIT INT TERM

umask 022
mkdir -p "$ADMIN_RELEASE" "$USER_RELEASE" "$STATE_DIR"
tar -xzf "$FRONTEND_STAGE/alpha-admin-web.tar.gz" -C "$ADMIN_RELEASE"
tar -xzf "$FRONTEND_STAGE/alpha-user-next.tar.gz" -C "$USER_RELEASE"
test -f "$ADMIN_RELEASE/index.html"
test -f "$USER_RELEASE/apps/user-next/server.js"

ADMIN_CONTEXT="$STAGING/admin-build"
USER_CONTEXT="$STAGING/user-build"
mkdir -p "$ADMIN_CONTEXT/docker/nginx" "$ADMIN_CONTEXT/src/frontend/apps/admin/dist"
mkdir -p "$USER_CONTEXT/app"
cp "$DEPLOY_STAGE/docker/frontend.dist.Dockerfile" "$ADMIN_CONTEXT/docker/frontend.dist.Dockerfile"
cp "$DEPLOY_STAGE/docker/frontend.user.Dockerfile" "$USER_CONTEXT/frontend.user.Dockerfile"
cp "$DEPLOY_STAGE/docker/nginx/frontend.conf" "$ADMIN_CONTEXT/docker/nginx/frontend.conf"
cp -R "$ADMIN_RELEASE/." "$ADMIN_CONTEXT/src/frontend/apps/admin/dist/"
cp -R "$USER_RELEASE/." "$USER_CONTEXT/app/"

docker build \
  --build-arg "VERSION=$RELEASE_ID" \
  --tag "alpha-server:candidate-$RELEASE_ID" \
  "$SERVER_STAGE"
docker build \
  --build-arg "VERSION=$RELEASE_ID" \
  --file "$USER_CONTEXT/frontend.user.Dockerfile" \
  --tag "alpha-frontend-user:candidate-$RELEASE_ID" \
  "$USER_CONTEXT"
docker build \
  --build-arg APP_NAME=admin \
  --build-arg "VERSION=$RELEASE_ID" \
  --file "$ADMIN_CONTEXT/docker/frontend.dist.Dockerfile" \
  --tag "alpha-frontend-admin:candidate-$RELEASE_ID" \
  "$ADMIN_CONTEXT"

docker compose --env-file "$ROOT/.env" -f "$DEPLOY_STAGE/docker-compose.yml" config --quiet

docker run -d --rm \
  --name "$USER_PREFLIGHT_CONTAINER" \
  --network alpha_default \
  --user bun \
  --read-only \
  --tmpfs /tmp:size=64m,mode=1777 \
  --tmpfs /app/apps/user-next/.next/cache:size=64m,uid=1000,gid=1000,mode=0750 \
  --env NODE_ENV=production \
  --env HOSTNAME=0.0.0.0 \
  --env PORT=3000 \
  --env NEXT_TELEMETRY_DISABLED=1 \
  --env NEXT_PUBLIC_ALPHA_DEMO=false \
  --env ALPHA_UPSTREAM_URL=http://server:27001 \
  --env ALPHA_UPSTREAM_API_PREFIX= \
  --env "ALPHA_SUBSCRIPTION_PROXY_PATH=$SUBSCRIBE_PATH" \
  --env ALPHA_AUTH_COOKIE_SECURE=false \
  --env "ALPHA_PUBLIC_ORIGIN=http://$LAN_HOST:3000" \
  "alpha-frontend-user:candidate-$RELEASE_ID" >/dev/null

preflight_deadline=$(( $(date +%s) + 60 ))
until docker exec "$USER_PREFLIGHT_CONTAINER" bun -e \
  "fetch('http://127.0.0.1:3000/login').then(r=>{if(!r.ok)process.exit(1)}).catch(()=>process.exit(1))"; do
  if [ "$(date +%s)" -ge "$preflight_deadline" ]; then
    docker logs "$USER_PREFLIGHT_CONTAINER" >&2 || true
    echo "user candidate preflight failed" >&2
    exit 1
  fi
  sleep 1
done
docker exec "$USER_PREFLIGHT_CONTAINER" bun -e \
  "fetch('http://127.0.0.1:3000/api/backend/v1/common/site/config').then(async r=>{if(!r.ok||!(await r.text()).includes('\"code\":200'))process.exit(1)}).catch(()=>process.exit(1))"
docker stop "$USER_PREFLIGHT_CONTAINER" >/dev/null

docker run -d --rm \
  --name "$ADMIN_PREFLIGHT_CONTAINER" \
  --network alpha_default \
  "alpha-frontend-admin:candidate-$RELEASE_ID" >/dev/null
preflight_deadline=$(( $(date +%s) + 60 ))
until docker exec "$ADMIN_PREFLIGHT_CONTAINER" wget -q -O /dev/null http://127.0.0.1/; do
  if [ "$(date +%s)" -ge "$preflight_deadline" ]; then
    docker logs "$ADMIN_PREFLIGHT_CONTAINER" >&2 || true
    echo "admin candidate preflight failed" >&2
    exit 1
  fi
  sleep 1
done
docker stop "$ADMIN_PREFLIGHT_CONTAINER" >/dev/null

umask 077
mkdir -p "$BACKUP_DIR"
chmod 700 "$BACKUP_DIR"
cp -p "$COMPOSE_FILE" "$BACKUP_DIR/docker-compose.yml"
cp -p "$ROOT/.env" "$BACKUP_DIR/env"
docker compose --env-file "$ROOT/.env" -f "$COMPOSE_FILE" exec -T postgres sh -c \
  'pg_dump -Fc -U "$POSTGRES_USER" "$POSTGRES_DB"' > "$BACKUP_DIR/postgres-before.dump"

service_image_id() {
  service=$1
  container_id=$(docker compose --env-file "$ROOT/.env" -f "$COMPOSE_FILE" ps -q "$service")
  if [ -z "$container_id" ]; then
    echo "running Compose service is missing: $service" >&2
    return 1
  fi
  docker inspect "$container_id" --format '{{.Image}}'
}

OLD_SERVER_IMAGE=$(service_image_id server)
OLD_USER_IMAGE=$(service_image_id user)
OLD_ADMIN_IMAGE=$(service_image_id admin)
printf '%s\n' "$OLD_SERVER_IMAGE" > "$BACKUP_DIR/server-image"
printf '%s\n' "$OLD_USER_IMAGE" > "$BACKUP_DIR/user-image"
printf '%s\n' "$OLD_ADMIN_IMAGE" > "$BACKUP_DIR/admin-image"

update_env() {
  name=$1
  value=$2
  temporary=$(mktemp "$ROOT/.env.release.XXXXXX")
  awk -v name="$name" -v value="$value" '
    BEGIN { found = 0 }
    index($0, name "=") == 1 { print name "=" value; found = 1; next }
    { print }
    END { if (!found) print name "=" value }
  ' "$ROOT/.env" > "$temporary"
  chmod 600 "$temporary"
  mv "$temporary" "$ROOT/.env"
}

activated=false
server_switched=false
rollback() {
  status=$?
  trap - EXIT INT TERM
  if [ "$activated" = true ]; then
    exit "$status"
  fi
  echo "release activation failed; restoring previous application release" >&2
  cp -p "$BACKUP_DIR/docker-compose.yml" "$COMPOSE_FILE"
  cp -p "$BACKUP_DIR/env" "$ROOT/.env"
  docker image tag "$OLD_USER_IMAGE" alpha-frontend-user:lan
  docker image tag "$OLD_ADMIN_IMAGE" alpha-frontend-admin:lan
  if [ "$server_switched" = true ]; then
    echo "database migration may have started; retaining the candidate server image" >&2
    docker image tag "alpha-server:candidate-$RELEASE_ID" alpha-server:lan
  else
    docker image tag "$OLD_SERVER_IMAGE" alpha-server:lan
  fi
  docker compose --env-file "$ROOT/.env" -f "$COMPOSE_FILE" up -d --no-deps --no-build --force-recreate server user admin || true
  exit "$status"
}
cleanup_candidate=false
trap rollback EXIT INT TERM

mkdir -p "$ROOT/docker/nginx"
cp "$DEPLOY_STAGE/docker-compose.yml" "$COMPOSE_FILE"
cp "$DEPLOY_STAGE/docker/frontend.dist.Dockerfile" "$ROOT/docker/frontend.dist.Dockerfile"
cp "$DEPLOY_STAGE/docker/frontend.user.Dockerfile" "$ROOT/docker/frontend.user.Dockerfile"
cp "$DEPLOY_STAGE/docker/nginx/frontend.conf" "$ROOT/docker/nginx/frontend.conf"
cp "$DEPLOY_STAGE/prepare-runtime-permissions.sh" "$ROOT/prepare-runtime-permissions.sh"
chmod 700 "$ROOT/prepare-runtime-permissions.sh"
"$ROOT/prepare-runtime-permissions.sh" "$ROOT"
update_env ALPHA_USER_PUBLIC_ORIGIN "http://$LAN_HOST:3000"
update_env ALPHA_AUTH_COOKIE_SECURE false
update_env ALPHA_AUTH_COOKIE_MAX_AGE 3600
update_env ALPHA_SUBSCRIPTION_PROXY_PATH "$SUBSCRIBE_PATH"

docker image tag "alpha-server:candidate-$RELEASE_ID" alpha-server:lan
docker image tag "alpha-frontend-user:candidate-$RELEASE_ID" alpha-frontend-user:lan
docker image tag "alpha-frontend-admin:candidate-$RELEASE_ID" alpha-frontend-admin:lan

server_switched=true
docker compose --env-file "$ROOT/.env" -f "$COMPOSE_FILE" up -d --no-deps --no-build --force-recreate server

deadline=$(( $(date +%s) + 180 ))
until curl -fsS --max-time 5 http://127.0.0.1:27001/v1/common/heartbeat | grep -q '"status":true'; do
  if [ "$(date +%s)" -ge "$deadline" ]; then
    echo "server heartbeat did not become healthy" >&2
    exit 1
  fi
  sleep 3
done

docker compose --env-file "$ROOT/.env" -f "$COMPOSE_FILE" up -d --no-deps --no-build --force-recreate user admin

deadline=$(( $(date +%s) + 180 ))
while :; do
  user_status=$(curl -sS --max-time 5 -o /dev/null -w '%{http_code}' http://127.0.0.1:3000/login || true)
  admin_status=$(curl -sS --max-time 5 -o /dev/null -w '%{http_code}' http://127.0.0.1:3001/ || true)
  site_config=$(curl -fsS --max-time 5 http://127.0.0.1:3000/api/backend/v1/common/site/config 2>/dev/null || true)
  if [ "$user_status" = 200 ] && [ "$admin_status" = 200 ] && printf '%s' "$site_config" | grep -q '"code":200'; then
    break
  fi
  if [ "$(date +%s)" -ge "$deadline" ]; then
    echo "frontend health or same-origin API proxy check failed" >&2
    exit 1
  fi
  sleep 3
done

origin_status=$(curl -sS --max-time 5 -o /dev/null -w '%{http_code}' \
  -H 'Content-Type: application/json' \
  --data '{"email":"probe@example.invalid","password":"not-a-real-password"}' \
  http://127.0.0.1:3000/api/auth/login)
if [ "$origin_status" != 403 ]; then
  echo "same-origin mutation guard returned HTTP $origin_status" >&2
  exit 1
fi

printf '%s\n' "$RELEASE_ID" > "$STATE_DIR/release.current"
docker image inspect alpha-frontend-user:lan --format '{{.Id}}' > "$STATE_DIR/user-next.current"
docker image rm "alpha-server:candidate-$RELEASE_ID" >/dev/null 2>&1 || true
docker image rm "alpha-frontend-user:candidate-$RELEASE_ID" >/dev/null 2>&1 || true
docker image rm "alpha-frontend-admin:candidate-$RELEASE_ID" >/dev/null 2>&1 || true
activated=true
trap - EXIT INT TERM

echo "release activated: $RELEASE_ID"
echo "database snapshot: $BACKUP_DIR/postgres-before.dump"
