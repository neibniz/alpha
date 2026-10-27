# syntax=docker/dockerfile:1.7@sha256:a57df69d0ea827fb7266491f2813635de6f17269be881f696fbfdf2d83dda33e

FROM --platform=$BUILDPLATFORM golang:1.27.3-alpine3.24@sha256:d200d2dd76b03a99b52651fd1325aff7dcd35422c35c01bac56571b0507f3deb AS builder

ARG TARGETOS
ARG TARGETARCH
WORKDIR /src
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download
COPY . .
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -trimpath -ldflags="-s -w" -o /out/forwared .

FROM alpine:3.24@sha256:28bd5fe8b56d1bd048e5babf5b10710ebe0bae67db86916198a6eec434943f8b

RUN apk add --no-cache ca-certificates tzdata \
    && addgroup -S -g 10001 forwared \
    && adduser -S -D -H -u 10001 -G forwared -h /app forwared \
    && mkdir -p /app/config /app/log /app/state \
    && chown -R forwared:forwared /app

WORKDIR /app
COPY --from=builder /out/forwared /usr/local/bin/forwared
COPY config/forwared.example.yaml /app/config/forwared.example.yaml

USER forwared
ENTRYPOINT ["/usr/local/bin/forwared"]
CMD ["-config", "/app/config/forwared.yaml"]
