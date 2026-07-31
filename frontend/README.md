# Alpha Frontend

The Alpha frontend is a Bun workspace with two production applications:

| Path | Runtime | Purpose |
| --- | --- | --- |
| `apps/admin` | Vite static build | Administration console |
| `apps/user-next` | Next.js standalone on Bun | User portal |

Shared UI and TypeScript configuration live under `packages/`.

## Development

Install the Bun version declared in `package.json`, then run commands from this
directory:

```bash
bun install --frozen-lockfile
bun run dev
bun run check
bun run test
bun run build
```

Run one application with a workspace filter:

```bash
bun --filter alpha-admin-web dev
bun --filter alpha-user-next dev
```

## Release

```bash
bash scripts/package-release.sh
```

The script produces `alpha-admin-web.tar.gz` and `alpha-user-next.tar.gz` in
`release/`. GitHub Actions publishes the same archives for `frontend/v*` tags.

Both applications can also be built as independent images:

```bash
docker compose -f compose.yml build
```

Production deployment is managed from [`../deploy`](../deploy/README.md).
Container image names, runtime variables, Vercel, and Cloudflare settings are documented in
[`DEPLOYMENT.md`](DEPLOYMENT.md).
