# Alpha LAN Docker Deployment

This bundle is intended for the LAN host deployment under `/data/alpha`.

It is an upgrade and verification bundle for an existing host. Initial production provisioning,
including Docker installation and secret generation, should use the Ansible playbooks in
`deploy/`. Before a manual LAN bootstrap, install `.env.example` as `/data/alpha/.env`, replace
every `CHANGE_ME` value, and set mode `0600`.

Public ports:

- User frontend: `http://<host>:3000`
- Admin frontend: `http://<host>:3001`
- Server API: `http://<host>:27001`

PostgreSQL, Redis, and the single-node ClickHouse instance are only reachable on the Compose network. ClickHouse persists data below `/data/alpha/data/clickhouse` and is not exposed on a host port.

The admin frontend uses same-origin `/api` calls through Nginx. The user frontend is the Next.js standalone application: its server-only BFF keeps the access token in an HttpOnly cookie and talks to the Go service over the Compose network.

Build the frontend release artifacts on the controller:

```sh
cd frontend
bash scripts/package-release.sh
```

Stage the two archives, their `.sha256` files, the server source, and the LAN deployment files below a unique `/data/alpha/.staging/<release>` directory. Then activate them with:

```sh
/data/alpha/activate-release.sh \
  /data/alpha \
  /data/alpha/.staging/<release> \
  <release> \
  192.168.31.192 \
  /api/subscribe
```

Activation verifies both checksums, builds separate candidate server, user, and admin images,
creates a PostgreSQL snapshot, and performs server, frontend, BFF, and same-origin checks. A failed
frontend health check restores the previous frontend images and configuration. Once the candidate
Server has started, it is retained on failure because a database migration may already have run;
the PostgreSQL snapshot is retained for operator-led recovery and is never restored automatically
over a live database.

The final argument is the public subscription URL path configured in Admin. It defaults to `/api/subscribe`, must be an absolute path, and can be changed on the next activation without rebuilding either frontend.

User release inputs live below `/data/alpha/releases/user-next` and are baked into a candidate Bun image before activation. Admin static files are baked into a separate candidate Nginx image. Neither frontend container depends on application files mounted from the host.

Remote layout:

```text
/data/alpha
  .env
  docker-compose.yml
  docker/
  config/
  data/
  releases/
  secrets/
  src/
```

`/data/alpha/config/server/alpha.yaml`, `/data/alpha/.env`, and `/data/alpha/secrets/privacy-keyring.json` contain deployment secrets and are generated per host. Back up the privacy keyring separately; losing it makes encrypted user identities and access records unrecoverable.

The `.env` file must define `ALPHA_CLICKHOUSE_PASSWORD`; `ALPHA_CLICKHOUSE_USER` and `ALPHA_CLICKHOUSE_DB` default to `alpha` and `alpha_analytics`.

For an existing `/data/alpha` deployment, run `./configure-clickhouse.sh /data/alpha` once before starting the updated Compose stack. It preserves existing PostgreSQL, Redis, and application secrets, creates a ClickHouse password when missing, and enables the server analytics connection.

Run `./configure-privacy.sh /data/alpha` once before starting the updated server. It creates the AES-GCM and blind-index keys with mode `0600` when absent, preserves an existing keyring, and enables the privacy section in the server config.

Run `./configure-runtime-security.sh /data/alpha <LAN-host>` during the node-authentication cutover. It rotates the node master and deployment credentials, writes the deployment token to `/data/alpha/secrets/node-deploy-token`, configures the LAN frontend origin allowlist, clears the node configuration cache, and verifies that legacy query-string authentication is rejected.

After the cutover, `./verify-security.sh /data/alpha` validates credential file modes, legacy-secret removal, CORS boundaries, Admin authentication, runtime login-log redaction, secret-free node settings, per-server node-token binding, and cleanup of its test servers.

After deployment, `./verify-analytics.sh /data/alpha` validates gzip ingestion with partial row rejection, Admin-authenticated querying, masking, blind indexes, traffic outbox delivery, and ClickHouse persistence, then removes its synthetic events.

`./verify-traffic-series.sh verify /data/alpha` creates an isolated subscription with 12 ClickHouse traffic samples, verifies the one-hour Admin and User series plus ownership isolation, and removes the fixture. Use the `setup`, `check`, and `cleanup` actions when the fixture must remain available for browser testing.

After building `alpha-xray-agent:lan`, `./verify-xray-agent-access.sh /data/alpha` starts an isolated temporary agent, verifies threshold-triggered spool delivery into ClickHouse, and removes its temporary server and access records.
