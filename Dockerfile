# syntax=docker/dockerfile:1

FROM golang:1.27.2-bookworm AS builder
WORKDIR /src

COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download && go mod verify

COPY cmd ./cmd
COPY internal ./internal
COPY migrations ./migrations
# The WebP encoder includes its C sources and requires CGO and a C compiler.
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=1 go test ./... && \
    CGO_ENABLED=1 go build -trimpath -ldflags="-s -w" -o /out/server ./cmd/server

FROM debian:bookworm-slim
RUN apt-get update && \
    apt-get install -y --no-install-recommends ca-certificates curl && \
    rm -rf /var/lib/apt/lists/* && \
    useradd --system --uid 10001 --user-group pixstore && \
    mkdir -p /app/storage && chown pixstore:pixstore /app/storage

WORKDIR /app
COPY --from=builder /out/server /app/server
ENV PORT=8080 STORAGE_PATH=/app/storage
USER pixstore
EXPOSE 8080
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD curl --fail --silent --output /dev/null "http://127.0.0.1:${PORT}/health" || exit 1
ENTRYPOINT ["/app/server"]
