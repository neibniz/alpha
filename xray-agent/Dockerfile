# syntax=docker/dockerfile:1.7@sha256:a57df69d0ea827fb7266491f2813635de6f17269be881f696fbfdf2d83dda33e

ARG XRAY_DATA_IMAGE=ghcr.io/xtls/xray-core@sha256:592ec4d11f656db95598d01e76dbcc6e002d67360b96a5436500a938230f52c7

FROM ${XRAY_DATA_IMAGE} AS xray-data

FROM --platform=$BUILDPLATFORM golang:1.27.3-alpine@sha256:d200d2dd76b03a99b52651fd1325aff7dcd35422c35c01bac56571b0507f3deb AS builder

ARG TARGETOS
ARG TARGETARCH
WORKDIR /src
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
  go mod download
COPY . .
RUN --mount=type=cache,target=/go/pkg/mod \
  --mount=type=cache,target=/root/.cache/go-build \
  CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -trimpath -ldflags="-s -w" -o /out/xray-agent ./cmd/xray-agent \
  && CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -trimpath -ldflags="-s -w" -o /out/xray github.com/xtls/xray-core/main

FROM alpine:3.23.5@sha256:fd791d74b68913cbb027c6546007b3f0d3bc45125f797758156952bc2d6daf40

RUN apk add --no-cache ca-certificates tzdata curl unzip \
  && addgroup -S alpha \
  && adduser -S -G alpha alpha \
  && mkdir -p /var/lib/alpha/xray-agent /etc/alpha

COPY --from=builder /out/xray-agent /usr/local/bin/xray-agent
COPY --from=builder /out/xray /usr/local/bin/xray
COPY --from=xray-data /usr/local/share/xray /usr/local/share/xray
COPY configs/example.yaml /etc/alpha/xray-agent.example.yaml

VOLUME ["/var/lib/alpha/xray-agent", "/etc/alpha"]

ENTRYPOINT ["/usr/local/bin/xray-agent"]
CMD ["-config", "/etc/alpha/xray-agent.yaml"]
