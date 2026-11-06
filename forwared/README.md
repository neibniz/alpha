# Forwared

TCP forwarding service with automatic Server registration, live metrics, and
batched connection events. UDP forwarding is not supported.

## Docker Compose

Only one persistent volume is needed. Configuration comes from environment
variables; logs go to stdout and Docker rotates them. No host database directory
or configuration-file mount is required.

```sh
cp .env.example .env
# Configure the API origin, Server bootstrap secret, and database download source.
chmod 600 .env
docker compose pull
docker compose up -d
docker compose logs --tail=100 -f
```

The Compose file uses `ghcr.io/neibniz/alpha-forwared:latest`. Pin a published
release tag for production. New code requires a newly built/published image;
updating repository files does not change an existing registry image.

Required configuration:

```dotenv
ALPHA_FORWARED_BASE_URL=https://api.example.com
ALPHA_FORWARED_SECRET_KEY=<same-value-as-Server-Forwarder.BootstrapSecret>
ALPHA_FORWARED_IP2LOCATION_TOKEN=<your-IP2Location-download-token>
```

The API origin must not include `/v1`. The bootstrap secret must contain at least
32 characters. The IP2Location token is a different credential: obtain it from
your own [IP2Location account](https://www.ip2location.com/free/downloader).
Do not commit `.env`, tokens, downloaded databases, or persistent state.

Optional environment variables:

| Variable | Default / purpose |
| --- | --- |
| `ALPHA_FORWARED_NAME` | Display name in Admin |
| `ALPHA_FORWARED_PUBLIC_IP` | Explicit public IP; otherwise discover automatically |
| `ALPHA_FORWARED_DATA_DIR` | `/app/data` |
| `ALPHA_FORWARED_IP2LOCATION_CODE` | `DB3LITEBINIPV6` (IPv4 and IPv6, country/region/city) |
| `ALPHA_FORWARED_GEOIP_URL` | Trusted HTTPS ZIP/BIN mirror; overrides the official-token source |
| `ALPHA_FORWARED_GEOIP_UPDATE_INTERVAL` | `86400` seconds; allowed range: 1 hour to 30 days |

For a bind mount instead of a named volume, use `./data:/app/data` and make the
directory writable by UID/GID `10001`. Do not share one data volume between
different forwarder instances. Listen ports should be at least `1024`.

## Database Updates

- An empty installation downloads and validates the database before registering
  and accepting traffic. A missing token/source or failed download is an explicit
  startup error, not a silent bypass of the existing geographic policy.
- An existing valid database is loaded immediately, including when the download
  source is unavailable. Updates run in the background, normally once a day;
  failed updates are retried hourly and leave the old database untouched.
- HTTP conditional requests avoid unchanged downloads where supported. Downloads
  have a timeout and size limits; ZIP extraction cannot write archive paths.
  Invalid, truncated, unsupported, or older databases are rejected.
- A valid new reader is installed atomically without restarting listeners.
  Concurrent lookups cannot observe a closed or partially replaced reader.
- Files are stored under `/app/data`: `identity.json`,
  `connection-events.jsonl`, and `geoip/IP2LOCATION-LITE-DB3.BIN`.
- The image and Git repository do not contain the database. Comply with the
  database provider's license and download quotas. Forwared uses the
  [IP2Location LITE database for IP geolocation](https://www.ip2location.com).

Official downloads follow HTTPS redirects, including the provider's signed
storage URLs. Allow those destinations in the host's outbound firewall. Download
URLs, tokens, and error response bodies are not written to application logs.

## Existing Installations

Explicit `-config /path/to/forwared.yaml` still works. Without that flag,
`./forwared.yaml` and `/app/config/forwared.yaml` are discovered if present.
Environment variables take precedence over YAML. Existing explicit `geoipfile`,
`state_file`, `spool_file`, and `logfile` paths remain supported; a read-only
`geoipfile` cannot receive automatic updates.

Before replacing legacy `config/`, `log/`, and `state/` mounts with one data
volume, stop the old instance and copy its `state/identity.json`, pending
`state/connection-events.jsonl`, and database into the new layout. Keep the
original instance identity and never use `docker compose down -v` unless the
state is intentionally disposable. The Ansible role performs this copy only
when the destination does not already exist, preserving the original files.

Configure nodes in Admin as described in the [deployment guide](../deploy/README.md#forwared-deployment).
Rules are derived from node addresses resolving to this forwarder's public IP.
Enable Server ClickHouse storage for connection-event history.
