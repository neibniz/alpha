# Alpha Server and Frontend Deployment

This directory contains the production deployment path for Semaphore UI + Ansible:

- `server` runs as a Docker container.
- `postgres`, `redis`, and single-node `clickhouse` run as Docker containers before server starts.
- `server` is exposed through a shared Caddy container.
- `frontend/apps/user-next` and `frontend/apps/admin` can be deployed from GitHub Release assets or independent container images.
- Package mode serves Admin through Caddy and runs User in a Bun runtime. Image mode runs `alpha-frontend-admin` and `alpha-frontend-user` as separate containers behind Caddy.
- Dependencies, server, user frontend, and admin frontend can be deployed on one host or split across inventory groups.

## Topology Variables

Set these in Semaphore UI extra vars or environment variables:

```yaml
postgres_target: postgres
redis_target: redis
clickhouse_target: clickhouse
server_target: server
user_frontend_target: frontend
admin_frontend_target: frontend
```

Examples:

```yaml
# Everything on one machine
postgres_target: alpha_all_in_one
redis_target: alpha_all_in_one
clickhouse_target: alpha_all_in_one
server_target: alpha_all_in_one
user_frontend_target: alpha_all_in_one
admin_frontend_target: alpha_all_in_one

# Database/cache, server, and frontend split
postgres_target: postgres
redis_target: redis
clickhouse_target: clickhouse
server_target: server
user_frontend_target: frontend
admin_frontend_target: frontend

# User and admin split
postgres_target: server
redis_target: server
clickhouse_target: server
server_target: server
user_frontend_target: user_frontend
admin_frontend_target: admin_frontend
```

Every target host that needs HTTP/HTTPS gets one shared `alpha-caddy` container. Project roles only add site fragments under:

```text
/data/alpha/caddy/sites
```

This avoids Caddy config collisions when server and frontend are deployed on the same machine.

## Inventory Layout

Production uses one host inventory. A physical machine is declared once and then assigned to the
service groups it runs. This avoids duplicate inventory aliases for the same IP and keeps dynamic
xray-agent matching deterministic.

The committed inventory contains only reserved example domains and TEST-NET addresses. Never put
real domains, host addresses, account emails, or private-key names in tracked inventory files.
For a local deployment, copy `hosts.yml` to the ignored `hosts.local.yml` and place variable
overrides in the ignored `group_vars/all/99-production.local.yml`. Semaphore users should keep the
same values in protected inventory and environment settings.

For example, `99-production.local.yml` is the appropriate place for `server_domain`, `user_domain`,
`admin_domain`, `caddy_email`, and `ansible_ssh_private_key_file` overrides.

```text
inventories/prod/
  hosts.yml
  group_vars/all/
    00-global.yml
    10-topology.yml
    20-server.yml
    30-datastores.yml
    40-frontend.yml
    50-agents.yml
    vault.yml.example
```

Assign the same host to multiple groups for an all-in-one deployment. If SSH is not listening on
port 22, set `ansible_port` once on the host:

```yaml
all:
  hosts:
    alpha-01:
      ansible_host: 192.0.2.10
      ansible_user: root
      ansible_port: 2222
  children:
    server:
      hosts:
        alpha-01:
    frontend:
      hosts:
        alpha-01:
```

## Run From Semaphore

Run commands from the repository's `deploy` directory. Its `ansible.cfg` selects
`inventories/prod/hosts.yml` automatically. Validate inventory, playbooks, rendered service
configuration, Compose files, Caddy sites, and shell scripts before creating a Semaphore task:

When using an ignored local inventory, append
`-i inventories/prod/hosts.local.yml` to `ansible-inventory` and `ansible-playbook` commands.

GitHub's `Deployment validation` workflow overrides this with `inventories/prod/ci.yml`. The CI
inventory defines the required groups but contains no target hosts, so validation never connects
to a managed machine.

```bash
cd deploy
python -m pip install -r requirements-ci.txt
make check
ansible-inventory --graph
```

Server only:

```bash
ansible-playbook playbooks/deploy-server.yml
```

Frontend only:

```bash
ansible-playbook playbooks/deploy-frontend.yml
```

Xray agent only:

```bash
ansible-playbook playbooks/deploy-xray-agent.yml
```

Forwarder only:

```bash
ansible-playbook playbooks/deploy-forwared.yml \
  -e 'forwared_targets=["forwarder-01","forwarder-02"]'
```

All:

```bash
ansible-playbook playbooks/deploy-all.yml
```

In Semaphore UI, map secrets into protected extra vars or Ansible Vault. For local runs, copy
`inventories/prod/group_vars/all/vault.yml.example` to the ignored `vault.yml`, replace every
value, and run `ansible-vault encrypt` on it. Do not place plaintext secrets in the numbered
configuration files or pass secrets through command-line extra vars.

Each deploy play has a service tag, and shared prerequisites keep their own tags. Examples:

```bash
ansible-playbook playbooks/deploy-all.yml --tags postgres
ansible-playbook playbooks/deploy-all.yml --tags admin_frontend
ansible-playbook playbooks/deploy-server.yml --tags caddy
```

Available service tags are `postgres`, `redis`, `clickhouse`, `server`, `user_frontend`,
`admin_frontend`, `xray_agent`, and `forwared`. The preflight role always runs and rejects any
deployment root outside `/data`.

## Server Dependencies

`deploy-server.yml` starts dependencies before the server container. The dependency targets can be the same machine as the server or separate inventory hosts:

```text
docker -> postgres -> redis -> clickhouse -> caddy -> server
```

```yaml
postgres_target: postgres
redis_target: redis
clickhouse_target: clickhouse
server_target: server
```

Default containers:

```yaml
postgres_image: postgres:16-alpine
redis_image: redis:7-alpine
clickhouse_image: clickhouse/clickhouse-server:26.3-alpine
postgres_addr: 127.0.0.1:5432
redis_host: 127.0.0.1:6379
clickhouse_addr: 127.0.0.1:9000
```

Postgres, Redis, and ClickHouse publish only on their `*_bind_host` values. These default to
`127.0.0.1`, so dependency ports are not exposed by an all-in-one deployment. Server, Caddy, and
containerized frontends use host networking while binding their listeners to `127.0.0.1`.
Caddy reaches local services through loopback, which prevents bypassing HTTPS and keeps
forwarded client IP handling aligned with the default trusted-proxy list. The User BFF trusts
the same-host Caddy headers and overwrites `X-Alpha-Client-IP` before calling Server. Login,
registration, OAuth, subscription access, order audit, and request logs therefore share one
resolved client address.

Set all required secret variables from Semaphore secrets or Ansible Vault. The server role writes
them as mode `0600` files under `/data/alpha/server/secrets` and injects only `*_FILE` environment
references into the container. The generated `alpha.yaml` contains no passwords or tokens.
Configuration and secret files are owned by the Server container UID, while the surrounding
directories remain private. A changed configuration or secret forces a Server recreation before
the heartbeat check.

The privacy keyring must use the Server schema shown in `vault.yml.example`. Generate separate
32-byte Base64 values for every encryption key and the blind-index key:

```bash
openssl rand -base64 32
openssl rand -base64 32
```

The Server role validates the JSON structure and decoded key lengths before changing the running
container.

When Postgres or Redis run on separate machines, override the connection host values so the server container can reach them:

```yaml
postgres_host: 10.0.0.10
postgres_bind_host: 10.0.0.10
redis_host: 10.0.0.11:6379
redis_bind_host: 10.0.0.11
clickhouse_host: 10.0.0.12
clickhouse_bind_host: 10.0.0.12
```

`redis_host` must resolve to a writable Redis primary. The server writes sessions,
cache entries, realtime samples, and Asynq queue data to Redis DB 5, so a Redis
replica will produce `READONLY You can't write against a read only replica`.
This can also happen with a single Redis process if it was changed at runtime
with `REPLICAOF`/`SLAVEOF` or if `redis_host` points at a managed service's
read-only endpoint. The bundled Redis role enforces the managed container as a
standalone primary and verifies writes on both the configured Redis DB and DB 5.
Check a target Redis before deploying:

```bash
redis-cli -h 10.0.0.11 -p 6379 INFO replication | grep '^role:'
redis-cli -h 10.0.0.11 -p 6379 -n 5 SET alpha:deploy:redis-write-check 1 EX 30
```

## Postgres Replication

The Postgres role supports physical streaming replication. Replication is controlled explicitly by
`postgres_replication_enabled`. With the default `false` value, the role keeps the old single-node
behavior and skips hosts in `postgres_replicas`. To enable replication, set the switch to `true`,
put one host in `postgres_primary`, and put one or more hosts in `postgres_replicas`:

```yaml
all:
  children:
    postgres:
      children:
        postgres_primary:
          hosts:
            postgres-01:
              ansible_host: 10.0.0.10
              ansible_user: root
        postgres_replicas:
          hosts:
            postgres-02:
              ansible_host: 10.0.0.12
              ansible_user: root
              # Optional when the primary must allow a private source IP instead of ansible_host.
              postgres_replication_source_host: 10.0.0.12
```

Set the replication password from Semaphore secrets or Ansible Vault:

```yaml
postgres_replication_enabled: true
postgres_replication_password: "change-me"
```

When `postgres_replication_enabled` is `true`, the role automatically:

- starts the primary with WAL settings suitable for streaming replication;
- creates or updates the replication user;
- adds managed `pg_hba.conf` rules for replica hosts;
- creates physical replication slots for replicas;
- bootstraps empty replica data directories with `pg_basebackup -R`;
- starts each replica and verifies `pg_is_in_recovery()`.

The server should continue to connect to the primary:

```yaml
postgres_host: 10.0.0.10
postgres_addr: "{{ postgres_host }}:{{ postgres_port }}"
```

Replica hosts with existing standalone Postgres data are not overwritten. To intentionally rebuild
a replica from the primary, set this only after taking a backup:

```yaml
postgres_replication_recreate_replica: true
```

Useful replication variables:

```yaml
postgres_replication_user: replicator
postgres_replication_hba_method: scram-sha-256
postgres_replication_slot_prefix: alpha_replica
postgres_wal_keep_size: 1GB
postgres_max_wal_senders: 10
postgres_max_replication_slots: 10
```

## Frontend Deploy Flow

Package mode remains the default and deploys the applications differently:

- `alpha-admin-web.tar.gz` is extracted into a checksum-addressed release and served by Caddy
  through an atomic `current` symlink.
- `alpha-user-next.tar.gz` is a Next.js standalone release. It runs in a restricted Bun container behind Caddy.

The target host never checks out frontend source or installs Bun. Admin and User packages are
verified against their release `.sha256` assets and extracted only once into checksum-addressed
directories. The roles switch `current` symlinks instead of deleting live content. User activation
verifies `/login` and restores the previous release if the health check fails. Both roles retain
three releases by default.

Both applications also support image mode. GitHub Actions publishes separate
multi-architecture images built with Bun:

```yaml
frontend_next_deployment_mode: image
frontend_next_container_image: ghcr.io/neibniz/alpha-frontend-user:1.4.2

frontend_admin_deployment_mode: image
frontend_admin_container_image: ghcr.io/neibniz/alpha-frontend-admin:1.4.2
```

In image mode, Caddy proxies to the local frontend containers. The roles pull
the selected image, verify its health endpoint, and restore the previous local
image when activation fails. Use an immutable version or digest in production.
`frontend_package_enabled` is ignored for an application running in image mode.

The example image references must exist in GHCR before deployment. This repository does not
silently fall back to a local build when a package or image is missing. Pin release tags or image
digests in production; `latest` and `edge` are intended only for controlled rollout workflows.

The target machines are prepared by the `docker` role before Caddy or application containers start. It installs Docker Engine and the Docker Compose plugin on Debian/Ubuntu and RHEL-compatible systems when Docker is missing.

Pin the component-prefixed frontend release. A repository-wide `latest` release
may belong to another component in this monorepo:

```yaml
frontend_package_enabled: true
frontend_release_version: frontend/v1.4.2
frontend_release_base_url: https://github.com/neibniz/alpha/releases
frontend_next_keep_releases: 3
```

Pin a specific frontend version when needed:

```yaml
frontend_package_enabled: true
frontend_release_version: frontend/v1.4.3
```

For an already deployed release, runtime-only changes do not require another package download. Set:

```yaml
frontend_package_enabled: false
frontend_next_upstream_url: http://127.0.0.1:27001
frontend_next_api_prefix: ""
```

The user container is recreated with the new runtime environment. The admin app continues to use its generated `runtime-config.js`.

If a frontend and Server are intentionally placed on different hosts, set the frontend upstream
to the Server's private or HTTPS address. Keep the frontend listener behind its same-host Caddy
instance; do not expose the Bun or Nginx port directly.

Set the User BFF IP source for its entry platform:

```yaml
# Automatic detection works for Cloudflare, Vercel, Caddy, and NGINX.
frontend_next_client_ip_source: auto

# Explicit cloudflare, vercel, or forwarded modes remain available for diagnostics.
```

The Caddy site templates overwrite downstream `X-Forwarded-For` and `X-Real-IP` values with
Caddy's direct client address. Cloudflare's `CF-Connecting-IP` and the User BFF's
`X-Alpha-Client-IP` continue to Server unchanged; keep the origin behind those configured entry
points rather than exposing the Server listener directly.

### User Subscription URL Proxy

The user frontend builds subscription URLs from the user site domain when no dedicated
`subscribe_domain` is configured. The user Caddy site therefore proxies the subscription endpoint
to the server before the SPA fallback:

```yaml
frontend_next_subscribe_proxy_enabled: true
frontend_next_subscribe_proxy_path: /v1/subscribe/config
frontend_next_subscribe_proxy_upstream: "127.0.0.1:{{ server_container_port }}"
```

If the user frontend and server run on different hosts, set the upstream to the server private
address:

```yaml
frontend_next_subscribe_proxy_upstream: 10.0.0.10:27001
```

Keep `frontend_next_subscribe_proxy_path` aligned with the server
`Subscribe.SubscribePath` value.

## Caddy Domains

Set:

```yaml
server_domain: api.example.com
user_domain: www.example.com
admin_domain: admin.example.com
frontend_api_base_url: https://api.example.com
```

`server_domain` reverse proxies to the local `alpha-server` container. `user_domain` reverse proxies to the local Next.js container, while `admin_domain` serves the admin static files with SPA fallback.

## Frontend Branding

Frontend favicon assets can be replaced after the release package is extracted, so changing them does not require rebuilding frontend bundles.

Default source directories:

```text
assets/branding/user/favicon
assets/branding/admin/favicon
```

Files in the matching directory are copied into the deployed frontend root, for example:

```text
/data/alpha/frontend/user/favicon.ico
/data/alpha/frontend/admin/favicon.ico
```

You can override the source per Semaphore task if needed:

```yaml
frontend_branding_favicon_src: "/path/to/favicon/files"
```

The Caddy site sets `no-cache` for favicon and web manifest files to make replacements visible quickly.

## Xray Agent Deployment

`deploy-xray-agent.yml` queries the server deployment API:

```text
GET /v2/server/deploy/servers
Authorization: Bearer <deployment-token>
```

The protected API returns each server's `id`, `name`, `address`, and server-bound token. The playbook matches the returned `address` against `ansible_host` values already configured in the Semaphore/Ansible inventory, skips API servers that are not registered as assets, and renders the matched server's `server_id` and token into `/data/alpha/xray-agent/config/xray-agent.yaml`.

Inventory example:

```yaml
xray-agent-01:
  ansible_host: 1.2.3.4 # Must match the address returned by /v2/server/deploy/servers.
  ansible_user: root
```

Required variables:

```yaml
xray_agent_deploy_token: "<node-deploy-secret-from-server-config>"
xray_agent_image: ghcr.io/neibniz/alpha-xray-agent:latest
xray_agent_server_base_url: "https://api.example.com"
```

## Forwared Deployment

`deploy-forwared.yml` deploys to a selected machine list rather than querying all servers. The selected values must match inventory hostnames or `ansible_host` IPs; unknown machines are skipped and the playbook stops if nothing matches. Pass a YAML list or a comma-separated string:

Inventory example:

```yaml
forwared-01:
  ansible_host: 5.6.7.8
  ansible_user: root
```

```yaml
forwared_targets:
  - forwarder-01
  - forwarder-02
```

The role renders `/data/alpha/forwared/config/forwared.yaml` and starts the configured forwarder image with host networking.

Required variables:

```yaml
server_forwarder_bootstrap_secret: "<dedicated-random-secret-at-least-32-bytes>"
forwared_control_token: "<same-dedicated-forwarder-secret>"
```

The bootstrap secret is used only to register a persistent forwarder instance. The server returns an instance-bound runtime token used for configuration, status and connection-event APIs. Public HTTPS is terminated by Caddy.

The GeoIP database can live in this deploy repository:

```text
assets/geoip/IP2LOCATION-LITE-DB3.BIN
```

Ansible copies the configured controller file on every run and restarts the process only when its
content changes. You can override the source or use a checksum-protected download:

```yaml
forwared_geoip_src: "/path/on/semaphore-runner/IP2LOCATION-LITE-DB3.BIN"
forwared_geoip_url: "https://example.com/IP2LOCATION-LITE-DB3.BIN"
forwared_geoip_checksum: "sha256:<64 lowercase or uppercase hexadecimal characters>"
```

`forwared_geoip_checksum` is mandatory when `forwared_geoip_url` is set. This prevents a mutable
remote file from being installed without integrity verification.

The forwarder container runs as UID/GID `10001` with all Linux capabilities removed. Configure
listen ports at `1024` or above; expose lower public ports through a host firewall redirect or
an external load balancer instead of running the forwarder as root.
