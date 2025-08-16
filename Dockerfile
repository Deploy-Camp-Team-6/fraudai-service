# syntax=docker/dockerfile:1.7

########################
# Build stage
########################
FROM golang:1.23-alpine AS builder

WORKDIR /app

# Install build deps (git often needed for private/modules)
RUN apk --no-cache add git

# Enable Go build caching across CI builds
# (BuildKit-only; harmless elsewhere)
ENV CGO_ENABLED=0 GOOS=linux

# Pre-copy go.mod/sum to leverage layer caching
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

# Copy the rest of the source
COPY . .

# Optional tooling installed in builder only (one layer)
RUN go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest && \
    go install github.com/golang-migrate/migrate/v4/cmd/migrate@latest

# Build with cache for speed
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go build -v -o /app/server ./cmd/api

########################
# Final runtime stage
########################
FROM alpine:3.20

# Only what we need at runtime
RUN apk --no-cache add ca-certificates

# Create non-root user
RUN addgroup -S app && adduser -S app -G app

WORKDIR /root

# Binaries and assets
COPY --from=builder /app/server ./server
COPY --from=builder /app/migrations ./migrations
# Ensure /root/api exists before copying file into it
RUN mkdir -p /root/api
COPY --from=builder /app/api/openapi.yaml /root/api/openapi.yaml
COPY --from=builder /app/config.yaml ./config.yaml
COPY --from=builder /go/bin/migrate /usr/local/bin/migrate
COPY --from=builder /app/entrypoint.sh ./entrypoint.sh

# Permissions
RUN chmod +x /root/entrypoint.sh && chown -R app:app /roeot

USER app

EXPOSE 8080

# Optional healthcheck if your server exposes /healthz locally
HEALTHCHECK --interval=10s --timeout=3s --retries=3 CMD wget -qO- http://127.0.0.1:8080/healthz >/dev/null || exit 1

ENTRYPOINT ["./entrypoint.sh"]
