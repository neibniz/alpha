# Frontend Platform Deployment

Alpha contains two independently deployable frontend applications:

| Application | Container image | Vercel | Cloudflare |
| --- | --- | --- | --- |
| Admin (`apps/admin`) | `alpha-frontend-admin` | Vite static project | Pages |
| User (`apps/user-next`) | `alpha-frontend-user` | Next.js project | Workers with OpenNext |

The user application is not a static Pages site. It uses SSR, Route Handlers,
OAuth callbacks, a BFF proxy, and HttpOnly authentication cookies. Cloudflare
supports those capabilities through Workers and the OpenNext adapter.

## Containers

The applications have separate multi-stage Dockerfiles and runtime images:

- `docker/user.Dockerfile` builds Next.js standalone with Bun and runs it with
  Bun as the unprivileged `bun` user.
- `docker/admin.Dockerfile` builds the Vite application with Bun and copies only
  static output into a restricted Nginx runtime.

Build them independently from the `frontend` directory:

```bash
docker build -f docker/user.Dockerfile -t alpha-frontend-user:local .
docker build -f docker/admin.Dockerfile -t alpha-frontend-admin:local .
```

Or build and start both applications:

```bash
docker compose -f compose.yml up -d --build
```

The Compose ports bind to `127.0.0.1` by default. Put a TLS reverse proxy in
front of them for production, or explicitly override `ALPHA_USER_BIND_HOST` and
`ALPHA_ADMIN_BIND_HOST`.

User runtime settings include:

```text
ALPHA_UPSTREAM_URL
ALPHA_UPSTREAM_API_PREFIX
ALPHA_SUBSCRIPTION_PROXY_PATH
ALPHA_AUTH_COOKIE_NAME
ALPHA_AUTH_COOKIE_MAX_AGE
ALPHA_AUTH_COOKIE_SECURE
ALPHA_PUBLIC_ORIGIN
ALPHA_TRUST_PROXY_HEADERS
ALPHA_CLIENT_IP_SOURCE
```

Admin defaults to same-origin `/api` calls. Set `ALPHA_ADMIN_UPSTREAM` to the
server origin reachable from its container. To use a public API origin instead,
pass `VITE_API_BASE_URL` and `VITE_API_PREFIX` as build arguments, or replace
`/usr/share/nginx/html/runtime-config.js` with a read-only runtime mount.

GitHub Actions publishes multi-architecture images to:

```text
ghcr.io/neibniz/alpha-frontend-user
ghcr.io/neibniz/alpha-frontend-admin
```

Commits receive `sha-*` tags, `main` receives `edge`, and
`frontend/vX.Y.Z` releases receive `X.Y.Z` and `latest`. Pin a version or digest
in production instead of relying on a mutable tag.

## Required Server Access

Both platforms must be able to reach the Alpha server through a public HTTPS
origin such as `https://api.example.com`.

The admin application talks directly to that origin, including its realtime
WebSocket endpoints. Add the exact admin production and preview origins to
`HTTP.AllowedOrigins` in the server configuration. Wildcard origins are not
accepted.

The user application talks to the server only from its Next.js server runtime,
so browser credentials remain on the user application's own origin.

### Client IP Forwarding

The User BFF overwrites `X-Alpha-Client-IP` with the address observed by its
deployment edge before each Server request. Select that edge explicitly with
`ALPHA_CLIENT_IP_SOURCE`:

| Deployment | Value |
| --- | --- |
| Cloudflare Workers | `cloudflare` |
| Vercel | `vercel` |
| Caddy or NGINX container | `forwarded` |
| Local development or automatic detection | `auto` |
| Disable forwarding | `none` |

No certificate or signing secret is required. Keep the Server listener behind
the configured reverse proxy, and make that proxy overwrite standard forwarding
headers instead of appending caller-provided values. Server request logs and all
authentication audit records then use the same resolved address.

## Vercel

Create two Vercel projects from the same repository.

### Admin

- Root Directory: `frontend/apps/admin`
- Framework Preset: Vite
- Build and install commands: read from `vercel.json`
- Required build variable: `VITE_API_BASE_URL=https://api.example.com`
- Optional build variable: `VITE_API_PREFIX` when the API uses a path prefix

### User

- Root Directory: `frontend/apps/user-next`
- Framework Preset: Next.js
- Build and install commands: read from `vercel.json`
- Required server variable: `ALPHA_UPSTREAM_URL=https://api.example.com`
- Required production variable: `ALPHA_PUBLIC_ORIGIN=https://user.example.com`
- Recommended variables:
  - `ALPHA_AUTH_COOKIE_SECURE=true`
  - `ALPHA_TRUST_PROXY_HEADERS=false`
  - `ALPHA_CLIENT_IP_SOURCE=vercel`
  - `NEXT_PUBLIC_ALPHA_DEMO=false`

For preview deployments without a fixed domain, omit `ALPHA_PUBLIC_ORIGIN` and
set `ALPHA_TRUST_PROXY_HEADERS=true`. Only use that setting on a trusted managed
proxy such as Vercel or Cloudflare.

For older Vercel projects, enable **Include source files outside the Root
Directory** because both applications import packages from `frontend/packages`.
New monorepo projects enable this by default.

## Cloudflare

Set `BUN_VERSION=1.3.14` for both Cloudflare projects so builds use the version
declared by the workspace.

### Admin on Pages

Create a Pages project with:

- Root directory: `frontend`
- Build command:
  `bun install --frozen-lockfile && bun --filter alpha-admin-web build`
- Build output directory: `apps/admin/dist`
- Required build variable: `VITE_API_BASE_URL=https://api.example.com`
- Optional build variable: `VITE_API_PREFIX` when the API uses a path prefix

The committed `_headers` and `wrangler.jsonc` files provide security headers,
asset caching, and direct Wrangler deployment. The application uses hash-based
routing, so Pages does not need a catch-all redirect:

```bash
cd frontend
bun run deploy:cloudflare:admin
```

### User on Workers

Create a Workers Builds project, not a static Pages project:

- Root directory: `frontend`
- Build command: `bun install --frozen-lockfile`
- Deploy command: `bun --filter alpha-user-next deploy:cloudflare`
- Required build variable and runtime variable:
  `ALPHA_UPSTREAM_URL=https://api.example.com`
- Required production runtime variable:
  `ALPHA_PUBLIC_ORIGIN=https://user.example.com`
- Recommended variables:
  - `ALPHA_AUTH_COOKIE_SECURE=true`
  - `ALPHA_TRUST_PROXY_HEADERS=false`
  - `ALPHA_CLIENT_IP_SOURCE=cloudflare`
  - `NEXT_PUBLIC_ALPHA_DEMO=false`

Configure the same non-public variables under both **Build Variables and
secrets** and the Worker's runtime variables when Cloudflare needs them during
the Next.js build. The deploy script uses `--keep-vars` so dashboard-managed
runtime values are not removed.

CLI preview and deployment are also available:

```bash
cd frontend
bun --filter alpha-user-next preview:cloudflare
bun run deploy:cloudflare:user
```

## OAuth

For GitHub OAuth, register each production callback URL:

```text
https://user.example.com/oauth/github/callback
```

Preview domains need separate GitHub OAuth applications or a stable preview
domain. Do not use a wildcard callback URL.

## Build-Time and Runtime Values

Admin `VITE_*` values are public and are written to `dist/runtime-config.js` at build
time. Container and Caddy deployments can still replace that file at runtime.

User variables without the `NEXT_PUBLIC_` prefix remain server-side. Never place
server credentials or OAuth client secrets in `NEXT_PUBLIC_*` variables.
