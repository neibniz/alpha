# syntax=docker/dockerfile:1.7

ARG BUN_IMAGE=oven/bun:1.3.14-slim@sha256:d56a2534ffd262e92c12fd3249d3924d296d97086da773f821d7d0477435ea04

FROM ${BUN_IMAGE} AS dependencies

WORKDIR /workspace

COPY package.json bun.lock bunfig.toml turbo.json tsconfig.json ./
COPY apps/admin/package.json apps/admin/package.json
COPY apps/user-next/package.json apps/user-next/package.json
COPY packages/typescript-config/package.json packages/typescript-config/package.json
COPY packages/ui/package.json packages/ui/package.json

RUN --mount=type=cache,target=/root/.bun/install/cache \
    bun install --frozen-lockfile --ignore-scripts

FROM dependencies AS builder

ARG NEXT_PUBLIC_ALPHA_DEMO=false
ARG ALPHA_SUBSCRIPTION_PROXY_PATH=/api/subscribe

ENV NEXT_TELEMETRY_DISABLED=1 \
    NEXT_PUBLIC_ALPHA_DEMO=${NEXT_PUBLIC_ALPHA_DEMO} \
    ALPHA_SUBSCRIPTION_PROXY_PATH=${ALPHA_SUBSCRIPTION_PROXY_PATH}

COPY . .

RUN bun --filter alpha-user-next build

FROM ${BUN_IMAGE} AS runtime

LABEL org.opencontainers.image.title="Alpha User" \
      org.opencontainers.image.description="Alpha user-facing Next.js frontend"

ENV NODE_ENV=production \
    HOSTNAME=0.0.0.0 \
    PORT=3000 \
    NEXT_TELEMETRY_DISABLED=1

WORKDIR /app

COPY --from=builder --chown=bun:bun /workspace/apps/user-next/.next/standalone/ ./
COPY --from=builder --chown=bun:bun /workspace/apps/user-next/.next/static/ ./apps/user-next/.next/static/
COPY --from=builder --chown=bun:bun /workspace/apps/user-next/public/ ./apps/user-next/public/

RUN mkdir -p /app/apps/user-next/.next/cache \
    && chown -R bun:bun /app

USER bun

EXPOSE 3000

HEALTHCHECK --interval=10s --timeout=5s --start-period=20s --retries=6 \
  CMD bun -e "fetch('http://127.0.0.1:3000/login').then(r=>{if(!r.ok)process.exit(1)}).catch(()=>process.exit(1))"

CMD ["bun", "apps/user-next/server.js"]
