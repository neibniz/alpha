# Deploy Alpha Server with Docker Compose

This guide defines the Linux single-host production baseline for Alpha Server,
PostgreSQL, Redis, and single-node ClickHouse. A host Caddy, NGINX, or equivalent
reverse proxy terminates TLS in front of Server.

The executable examples are:

- `server/docker-compose.yml`
- `server/etc/alpha.compose.example.yaml`
- `server/.env.compose.example`

Use the [Ansible deployment](../../deploy/README.md) for multi-host topology,
automated secret installation, or full Server, frontend, agent, and forwarder
deployment.

## Prepare

The target must be Linux on x86-64 or ARM64 with Docker Engine, Docker Compose
v2, and OpenSSL. Run from `server/`:

```bash
cp .env.compose.example .env
cp etc/alpha.compose.example.yaml etc/alpha.compose.yaml
install -d -m 0700 secrets
mkdir -p data/postgres data/redis data/clickhouse data/clickhouse-log
```

Edit `.env` and pin `ALPHA_SERVER_IMAGE` to an immutable release tag or digest.
Edit `etc/alpha.compose.yaml` and replace the example User/Admin origins and
administrator email. If a data-service host port is changed in `.env`, update
the corresponding Server connection address as well.

The example uses Linux host networking for Server. Server and all data services
listen only on host loopback, so a local reverse proxy is required for external
access. `HTTP.TrustedProxies` must contain only proxies that connect directly to
Server; never trust `0.0.0.0/0` or `::/0`.

## Create secrets

```bash
umask 077
openssl rand -hex 32 > secrets/jwt-access-secret
openssl rand -base64 32 > secrets/postgres-password
openssl rand -base64 32 > secrets/redis-password
openssl rand -base64 32 > secrets/clickhouse-password
openssl rand -hex 32 > secrets/node-secret
openssl rand -hex 32 > secrets/node-deploy-secret
openssl rand -hex 32 > secrets/forwarder-bootstrap-secret
openssl rand -base64 24 > secrets/admin-password

encryption_key="$(openssl rand -base64 32)"
index_key="$(openssl rand -base64 32)"
printf '{"active_key_id":"v1","encryption_keys":{"v1":"%s"},"index_key":"%s"}\n' \
  "$encryption_key" "$index_key" > secrets/privacy-keyring.json
unset encryption_key index_key
```

Store `secrets/admin-password` in a password manager, then run as root:

```bash
chown 65532:65532 secrets/*
chmod 0400 secrets/*
```

Server runs as UID/GID `65532:65532`, and mode `0400` both permits that
unprivileged process to read the mounted files and satisfies the privacy
keyring permission check. The data service entrypoints read their secrets as
root before dropping privileges. Back up the privacy keyring separately.
Losing an encryption key makes data encrypted with that key unrecoverable.

The administrator password is used only when the database contains no users;
changing the file later does not reset an existing administrator.

## Start and verify

```bash
docker compose config --quiet
docker compose pull
docker compose up -d
docker compose ps
docker compose logs --tail=200 server
curl -fsS http://127.0.0.1:8080/v1/common/heartbeat
```

Server starts after all three data services are healthy. On first startup it
runs the current PostgreSQL and ClickHouse migrations and creates the initial
administrator when there are no users. The number of migrated tables changes
with the release and is not a fixed deployment invariant.

A minimal Caddy site is:

```caddyfile
api.example.com {
    reverse_proxy 127.0.0.1:8080
}
```

The reverse proxy must overwrite untrusted forwarding headers. Server uses the
configured trusted-proxy chain for client addresses in authentication,
subscription access, and audit logs.

## Upgrade and backup

Back up PostgreSQL, `secrets/`, `.env`, and `etc/alpha.compose.yaml` before an
upgrade. Pin the new image, then run:

```bash
docker compose pull server
docker compose up -d --no-deps --force-recreate server
curl -fsS http://127.0.0.1:8080/v1/common/heartbeat
```

Migrations move forward. Confirm schema compatibility before rolling back only
the image; otherwise restore the pre-upgrade PostgreSQL backup.

Create a PostgreSQL logical backup with:

```bash
install -d -m 0700 backup
docker compose exec -T postgres pg_dump -U alpha -d alpha -Fc \
  > "backup/alpha-$(date +%Y%m%d-%H%M%S).dump"
```

Also back up ClickHouse data for access, traffic, and forwarder detail history.
Redis AOF contains sessions and pending asynchronous work but does not replace a
PostgreSQL backup. Test restores in an isolated environment.

## External data services

When using managed data services, remove their Compose services and matching
`depends_on` entries, change the addresses and TLS options in
`alpha.compose.yaml`, and retain the Server `ALPHA_*_PASSWORD_FILE` mounts.
Redis must be a writable primary; Server uses Redis DB 0 and DB 5. Keep all
database endpoints private or protected by strict firewall rules.
