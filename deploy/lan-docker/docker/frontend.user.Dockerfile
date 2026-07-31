ARG BUN_IMAGE=oven/bun:1.3.14-slim@sha256:d56a2534ffd262e92c12fd3249d3924d296d97086da773f821d7d0477435ea04

FROM ${BUN_IMAGE}

ARG VERSION=dev

LABEL org.opencontainers.image.title="Alpha User" \
      org.opencontainers.image.description="Alpha user-facing Next.js frontend" \
      org.opencontainers.image.version="${VERSION}"

ENV NODE_ENV=production \
    HOSTNAME=0.0.0.0 \
    PORT=3000 \
    NEXT_TELEMETRY_DISABLED=1

WORKDIR /app

COPY --chown=bun:bun app/ /app/

RUN test -f /app/apps/user-next/server.js \
    && mkdir -p /app/apps/user-next/.next/cache \
    && chown -R bun:bun /app

USER bun

EXPOSE 3000

HEALTHCHECK --interval=10s --timeout=5s --start-period=20s --retries=6 \
  CMD bun -e "fetch('http://127.0.0.1:3000/login').then(r=>{if(!r.ok)process.exit(1)}).catch(()=>process.exit(1))"

CMD ["bun", "apps/user-next/server.js"]
