# syntax=docker/dockerfile:1.7
#
# Hera slim image — pure Go binary, no Python runtime.
# Use this for chatbot-only deployments where the agent never needs
# to execute ad-hoc Python or install packages at runtime. Smallest
# footprint; no MCP python server support.
#
# For self-improving / multilingual deployments (python_exec,
# python_register_tool, etc.) use deployments/Dockerfile.full instead.
#
# Skills + configs are bundled at /app/bundled and exposed via
# HERA_BUNDLED. User data (config.yaml, hera.db, synced skills) lives
# in the /data volume and is populated on first boot by hera init.
#
# BuildKit cache mounts keep the Go module cache and Go build cache
# warm across builds. Enable BuildKit:
#   DOCKER_BUILDKIT=1 docker build -f deployments/Dockerfile .
# Docker Desktop and recent Docker CLI already default it on.
#
# Stage 1: Build
FROM golang:1.26-alpine AS builder

RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /app

# Cache dependencies. Copy manifests first so source-only edits don't
# invalidate the module download layer.
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

# Copy source.
COPY . .

# Build binaries. Cache mounts reuse the Go build cache between
# builds — a one-line source change rebuilds in <5s on a warm cache.
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /bin/hera ./cmd/hera

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /bin/hera-agent ./cmd/hera-agent

# Stage 2: Runtime
FROM alpine:3.19

RUN apk add --no-cache ca-certificates tzdata

# Create non-root user.
RUN addgroup -S hera && adduser -S hera -G hera

# Copy binaries.
COPY --from=builder /bin/hera /usr/local/bin/hera
COPY --from=builder /bin/hera-agent /usr/local/bin/hera-agent

# Bundled read-only reference tree: skills + example configs.
# hera init seeds $HERA_HOME from here on first boot.
COPY --from=builder /app/skills  /app/bundled/skills
COPY --from=builder /app/configs /app/bundled/configs

# Copy the entrypoint that runs hera init then exec hera-agent.
COPY deployments/docker-entrypoint.sh /usr/local/bin/docker-entrypoint.sh
RUN chmod +x /usr/local/bin/docker-entrypoint.sh

# HERA_BUNDLED: read-only reference tree inside the image.
ENV HERA_BUNDLED=/app/bundled

# HERA_HOME: user data volume (config, db, synced skills, logs).
# Defaults to /data; operators can override with -e HERA_HOME=...
ENV HERA_HOME=/data

# /data is the user data volume. Mount a named or host volume here
# to persist config, memory, and skill edits across restarts.
RUN mkdir -p /data && chown -R hera:hera /data /app/bundled
VOLUME /data

USER hera

EXPOSE 8080

ENTRYPOINT ["/usr/local/bin/docker-entrypoint.sh"]
