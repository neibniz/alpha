# Alpha Xray Agent

`xray-agent` runs on node servers. It is a standalone process that pulls Xray configuration from Alpha, writes a runtime `current.json`, starts and supervises `xray-core`, and reports realtime node status back to Alpha through WebSocket.

## Architecture

```text
Alpha server <----HTTP/WS----> xray-agent ----process----> xray-core
```

The agent owns control-plane work:

- Pull `/v2/server/{server_id}/xray-config`.
- Inject local Xray API, stats and policy blocks when enabled.
- Pull `/v2/server/{server_id}/xray-users` and inject effective users into inbound `settings.users`.
- Validate and write `current.json`.
- Start or restart `xray-core` when the rendered structure changes.
- Reconcile VLESS, VMess, Trojan and Shadowsocks inbound users through Xray `HandlerService` when only the effective user list changes.
- Report CPU, memory, disk, network speed, connection count and xray running state to `/v1/server/status/ws`.
- Collect user traffic through Xray `StatsService` and report deltas to `/v1/server/push`.
- Tail the Xray access log into a durable local spool and upload gzip batches to `/v1/server/access-logs`.

Access events are acknowledged locally only after the server confirms the ClickHouse insert. The collector flushes when either `batch_size` is reached or `flush_interval` expires, retries failed uploads with exponential backoff, and drains renamed log files before switching to the new active file.

When a product has `device_limit > 0`, the server uses these accepted access events to maintain an atomic active-network lease in Redis and returns scoped block decisions. The agent applies those decisions through Xray `RoutingService`, ahead of the configured routing rules, using a managed blackhole outbound. IPv4 is counted per `/32` and IPv6 per `/64`; Redis stores only blind indexes. This requires `xray.inject_api` and `report.access.enabled`. Because access logs are emitted after connection acceptance, a decision blocks subsequent connections and does not terminate the connection that produced the event.

The agent still keeps safe restart as the fallback path. Template, listener, transport, routing, DNS, policy or API changes restart `xray-core`; effective user changes are applied through `HandlerService` and the full runtime config is persisted to disk after a successful hot reload.

## Build

```bash
go build -o xray-agent ./cmd/xray-agent
```

## Run

```bash
./xray-agent -config ./configs/example.yaml
```

Environment variables can override common fields:

```bash
ALPHA_SERVER_URL=http://127.0.0.1:27001
ALPHA_SERVER_ID=1
ALPHA_NODE_TOKEN=alpha-node-v1_replace-with-derived-node-token
XRAY_BIN=/usr/local/bin/xray
XRAY_API_ADDRESS=127.0.0.1:10085
XRAY_AGENT_WORK_DIR=/var/lib/alpha/xray-agent
```

The preferred config format is YAML. Only the `server` block is required:

```yaml
server:
  base_url: "https://panel.example.com"
  server_id: 1
  token: "alpha-node-v1_replace-with-derived-node-token"
```

Everything else has safe defaults:

```yaml
xray:
  executable: "/usr/local/bin/xray"
  work_dir: "/var/lib/alpha/xray-agent"
  config_path: "/var/lib/alpha/xray-agent/current.json"
  last_good_path: "/var/lib/alpha/xray-agent/last_good.json"
  stats_cursor_path: "/var/lib/alpha/xray-agent/stats_cursor.json"
  api_address: "127.0.0.1:10085"
  inject_api: true
  validate_config: true

agent:
  pull_interval: "30s"

report:
  status_interval: "5s"
  traffic_interval: "30s"
  interface_name: ""
  access:
    enabled: true
    batch_size: 1000
    flush_interval: "5s"
    poll_interval: "1s"
    max_batch_bytes: 1048576
    max_spool_bytes: 268435456
```

Most deployments should keep these defaults. Override them only when xray-core is installed in a custom path, the runtime directory must be moved, or the local Xray API port conflicts with another process.

For compatibility, old JSON config files and legacy `panel` / `bin_path` / `config_file` field names can still be read.

## Docker

```bash
docker build -t alpha-xray-agent:local .
docker run -d --name alpha-xray-agent --restart unless-stopped \
  --network host \
  -v /etc/alpha/xray-agent.yaml:/etc/alpha/xray-agent.yaml:ro \
  -v /var/lib/alpha/xray-agent:/var/lib/alpha/xray-agent \
  alpha-xray-agent:local
```

The Docker image starts with `-config /etc/alpha/xray-agent.yaml` by default. To use environment variables only, override the default command with an empty argument list is not supported by Docker, so pass any supported flag explicitly or mount the minimal YAML config.

```bash
docker run -d --name alpha-xray-agent --restart unless-stopped \
  --network host \
  -v /var/lib/alpha/xray-agent:/var/lib/alpha/xray-agent \
  -e ALPHA_SERVER_URL=https://panel.example.com \
  -e ALPHA_SERVER_ID=1 \
  -e ALPHA_NODE_TOKEN=alpha-node-v1_replace-with-derived-node-token \
  alpha-xray-agent:local -config ""
```

The image builds both `xray-agent` and `xray-core` from the versions pinned in `go.mod` with the pinned Go toolchain. The separate upstream image supplies only geodata and is pinned by digest. When updating that data image, use another immutable digest:

```bash
docker build --build-arg XRAY_DATA_IMAGE=ghcr.io/xtls/xray-core@sha256:<digest> -t alpha-xray-agent:local .
```

## Next Milestones

- Add hot update for inbound/outbound/routing when the Xray API can apply the change safely.
- Keep safe restart as fallback for DNS, policy, API and transport listener changes.
