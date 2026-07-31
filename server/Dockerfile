# syntax=docker/dockerfile:1.7@sha256:a57df69d0ea827fb7266491f2813635de6f17269be881f696fbfdf2d83dda33e

# Use a smaller base image for the build stage
FROM --platform=$BUILDPLATFORM golang:1.26.5-alpine@sha256:0178a641fbb4858c5f1b48e34bdaabe0350a330a1b1149aabd498d0699ff5fb2 AS builder

LABEL stage=gobuilder

ARG TARGETOS
ARG TARGETARCH
ENV CGO_ENABLED=0

RUN apk add --no-cache tzdata ca-certificates curl

WORKDIR /build

# Download and verify the versioned IP database before copying application source
# so BuildKit can reuse this layer independently from normal code changes.
COPY assets/geoip/dbip-city-lite.env ./assets/geoip/dbip-city-lite.env
COPY script/fetch-geoip-database.sh ./script/fetch-geoip-database.sh
RUN sh ./script/fetch-geoip-database.sh /build/.bundled/dbip-city-lite.mmdb

# Copy go.mod and go.sum first to take advantage of Docker caching
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download

# Copy the rest of the application code
COPY . .

# Build the binary with version and build time
ARG VERSION
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    BUILD_TIME=$(date -u +"%Y-%m-%d %H:%M:%S") && \
    GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build \
      -ldflags="-s -w -X 'github.com/neibniz/alpha/server/pkg/constant.Version=${VERSION}' -X 'github.com/neibniz/alpha/server/pkg/constant.BuildTime=${BUILD_TIME}'" \
      -o /app/alpha .

# Final minimal image
FROM scratch

# Copy CA certificates and timezone data
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /usr/share/zoneinfo/Asia/Shanghai /usr/share/zoneinfo/Asia/Shanghai

ENV TZ=Asia/Shanghai \
    ALPHA_GEOIP_ENABLED=true \
    ALPHA_GEOIP_DATABASE_PATH=/usr/share/alpha/geoip/dbip-city-lite.mmdb

# Set working directory and copy binary
WORKDIR /app

COPY --from=builder /app/alpha /app/alpha
COPY --from=builder /build/etc/alpha.example.yaml /app/etc/alpha.yaml
COPY --from=builder /build/.bundled/dbip-city-lite.mmdb /usr/share/alpha/geoip/dbip-city-lite.mmdb
COPY --from=builder /build/THIRD_PARTY_NOTICES.md /usr/share/doc/alpha/THIRD_PARTY_NOTICES.md

# Expose the port (optional)
EXPOSE 8080

USER 65532:65532

# Specify entry point
ENTRYPOINT ["/app/alpha"]
CMD ["run", "--config", "etc/alpha.yaml"]
