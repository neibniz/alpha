# Alpha

Alpha is maintained as a monorepo. Each component keeps its own build system and
deployment boundary while sharing one Git history and one GitHub repository.

## Components

| Path | Purpose |
| --- | --- |
| `server/` | Go API server, workers, and database migrations |
| `frontend/` | Bun workspace containing the user and admin applications |
| `deploy/` | Ansible inventory, roles, and production playbooks |
| `xray-agent/` | Xray node agent and access-log collector |
| `forwared/` | TCP forwarding service and telemetry agent |

## Development

Run commands from the component directory:

```bash
cd server && go test ./...
cd frontend && bun install --frozen-lockfile && bun run build
cd xray-agent && go test ./...
cd forwared && go test ./...
cd deploy && make check
```

## Deployment

Choose one deployment path and keep its configuration as the source of truth:

| Scenario | Guide |
| --- | --- |
| Server with PostgreSQL, Redis, and ClickHouse on one Linux host | [Server Docker Compose](server/doc/docker-compose.md) / [中文](server/doc/docker-compose-zh.md) |
| Full stack or multi-host production deployment under `/data` | [Ansible deployment](deploy/README.md) |
| User/Admin containers, Vercel, or Cloudflare | [Frontend deployment](frontend/DEPLOYMENT.md) |

The Server Compose example is the shortest production-oriented path. Ansible is
recommended when Server, frontends, agents, forwarders, or data services need
to be managed across multiple hosts.

## Releases

Component releases use path-prefixed tags:

- `server/vX.Y.Z`
- `frontend/vX.Y.Z`
- `xray-agent/vX.Y.Z`
- `forwared/vX.Y.Z`

GitHub Actions use path filters for branch and pull-request builds. A component
tag starts only that component's release path.

Release tags must use the exact `component/vX.Y.Z` format, be newly created, and
point to a commit contained in the repository's default branch. Manual workflow
runs never create official releases. Release assets are built and verified
before versioned container images and the GitHub Release are published, and an
already published release is never overwritten.

The repository must also have a tag ruleset that blocks updates and deletions
for `server/v*`, `frontend/v*`, `xray-agent/v*`, and `forwared/v*`. Workflow
checks provide defense in depth, but cannot protect tags whose commits contain
an older workflow definition.

Component workflows publish consistently named container images:

- `ghcr.io/neibniz/alpha-server`
- `ghcr.io/neibniz/alpha-xray-agent`
- `ghcr.io/neibniz/alpha-frontend-user`
- `ghcr.io/neibniz/alpha-frontend-admin`
- `ghcr.io/neibniz/alpha-forwared`
