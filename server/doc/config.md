# Alpha Runtime Configuration

Alpha reads YAML from `./etc/alpha.yaml` by default. Use
`alpha run --config /path/to/alpha.yaml` to select another file.

The loader applies defaults automatically. Production configuration should declare only connection
details and behavior that differs from those defaults.

## Minimal Configuration

```yaml
HTTP:
  AllowedOrigins:
    - "https://user.example.com"
    - "https://admin.example.com"
  TrustedProxies:
    - "127.0.0.1"
    - "::1"

Postgres:
  Addr: "127.0.0.1:5432"
  Username: "alpha"
  Dbname: "alpha"

Redis:
  Host: "127.0.0.1:6379"

Administrator:
  Email: "admin@example.com"
```

The default listener is `0.0.0.0:8080`, production mode is enabled, and JWT access tokens expire
after seven days. Add `Host`, `Port`, `Debug`, or `JwtAuth.AccessExpire` only when overriding them.

## Client IP Resolution

Alpha resolves the client IP once, before request logging and route handlers run. Login, register,
OAuth, password reset, logout, order audit, subscription access, and forwarder registration all use
that same value.

- `TrustedProxies` lists only the reverse proxies that connect directly to Server. Never use
  `0.0.0.0/0` or `::/0`.
- `X-Forwarded-For` is evaluated from right to left across the trusted proxy chain; `X-Real-IP`
  is used as a fallback when that chain is absent.
- A User BFF running on Cloudflare, Vercel, or another host copies its observed address into
  `X-Alpha-Client-IP`. The BFF always overwrites a caller-supplied value before contacting Server.
- Cloudflare's `CF-Connecting-IP` is used when the BFF header is absent. Keep the origin behind
  Cloudflare so callers cannot bypass the edge and submit that header directly.

The BFF header intentionally requires no certificate or shared secret. Keep the Server listener
private behind the configured proxy and make every edge overwrite its source header. The resulting
address is the public address observed by the first edge; NATs and VPNs cannot reveal a device's
underlying address.

## Optional Components

Enable ClickHouse for access logs and traffic detail storage:

```yaml
ClickHouse:
  Enabled: true
  Addr: "127.0.0.1:9000"
  Database: "alpha_analytics"
  Username: "alpha"
```

Enable private-field encryption:

```yaml
Privacy:
  Enabled: true
  KeyFile: "/run/secrets/alpha_privacy_keyring"
```

For file logging inside a container, override only the mounted path and required policy:

```yaml
Logger:
  Mode: file
  Path: /app/log
  Level: info
```

## Runtime Secrets

Keep sensitive values out of `alpha.yaml`. Supply them as environment variables or read-only files:

| Configuration field | Environment variable |
|---|---|
| `JwtAuth.AccessSecret` | `ALPHA_JWT_ACCESS_SECRET` |
| `Postgres.Password` | `ALPHA_POSTGRES_PASSWORD` |
| `Redis.Pass` | `ALPHA_REDIS_PASSWORD` |
| `ClickHouse.Password` | `ALPHA_CLICKHOUSE_PASSWORD` |
| `Node.NodeSecret` | `ALPHA_NODE_SECRET` |
| `Node.NodeDeploySecret` | `ALPHA_NODE_DEPLOY_SECRET` |
| `Forwarder.BootstrapSecret` | `ALPHA_FORWARDER_BOOTSTRAP_SECRET` |
| `Administrator.Password` | `ALPHA_ADMIN_PASSWORD` |

Every variable supports a matching `_FILE` form:

```bash
ALPHA_POSTGRES_PASSWORD_FILE=/run/secrets/postgres-password
ALPHA_ADMIN_PASSWORD_FILE=/run/secrets/admin-password
```

Do not set both forms for the same secret. Alpha refuses to start when a configured secret file is
missing or empty. Sensitive YAML fields remain readable for backward compatibility, but should not
be used for new deployments.

## Deployment Requirements

- Set ordinary configuration and secret mounts to `0600`. The Docker Compose
  example sets its secret files to owner `65532:65532` and mode `0400` so the
  unprivileged Server can read them without allowing group or other access.
- Use at least 32 random characters for JWT and node secrets.
- Configure both node secrets together; leaving both empty disables node-authenticated operations.
- Use at least 20 random characters for PostgreSQL, Redis, and ClickHouse passwords.
- Do not emit secrets in logs, Compose output, or version control.
- The Ansible deployment renders a minimal `alpha.yaml` and mounts secrets through `_FILE` variables.
