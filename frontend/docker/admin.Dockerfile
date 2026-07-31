# syntax=docker/dockerfile:1.7

ARG BUN_IMAGE=oven/bun:1.3.14-slim@sha256:d56a2534ffd262e92c12fd3249d3924d296d97086da773f821d7d0477435ea04
ARG NGINX_IMAGE=nginx:1.30.3-alpine@sha256:0d3b80406a13a767339fbe2f41406d6c7da727ab89cf8fae399e81f780f814d1

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

ARG VITE_API_BASE_URL=
ARG VITE_API_PREFIX=/api
ARG VITE_AUTH_COOKIE_NAME=alpha_admin_authorization

ENV VITE_API_BASE_URL=${VITE_API_BASE_URL} \
    VITE_API_PREFIX=${VITE_API_PREFIX} \
    VITE_AUTH_COOKIE_NAME=${VITE_AUTH_COOKIE_NAME}

COPY . .

RUN bun --filter alpha-admin-web build

FROM ${NGINX_IMAGE} AS runtime

LABEL org.opencontainers.image.title="Alpha Admin" \
      org.opencontainers.image.description="Alpha administration frontend"

ENV ALPHA_ADMIN_BIND=0.0.0.0 \
    ALPHA_ADMIN_PORT=80 \
    ALPHA_ADMIN_UPSTREAM=http://host.docker.internal:27001

COPY docker/nginx/default.conf.template /etc/nginx/templates/default.conf.template
COPY --from=builder /workspace/apps/admin/dist/ /usr/share/nginx/html/

EXPOSE 80

HEALTHCHECK --interval=10s --timeout=5s --start-period=10s --retries=6 \
  CMD wget -q -O /dev/null "http://127.0.0.1:${ALPHA_ADMIN_PORT}/healthz" || exit 1
