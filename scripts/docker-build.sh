#!/usr/bin/env bash
# Hera Docker build helper.
#
# Usage:
#   scripts/docker-build.sh            # slim image (default)
#   scripts/docker-build.sh full       # full image with Python MCP
#   scripts/docker-build.sh slim       # explicit slim
#   scripts/docker-build.sh both       # build both variants in sequence
#
# Uses BuildKit cache mounts defined in the Dockerfiles so repeated
# builds reuse the Go module cache, Go build cache, apt cache, and
# pip cache. First cold build is the same as always; every subsequent
# one finishes in seconds.
set -euo pipefail

VARIANT="${1:-slim}"
export DOCKER_BUILDKIT=1
export COMPOSE_DOCKER_CLI_BUILD=1

here="$(cd "$(dirname "$0")/.." && pwd)"
cd "$here"

build_slim() {
  echo ">>> building hera:slim (${here}/deployments/Dockerfile)"
  time docker build \
    -f deployments/Dockerfile \
    -t hera:slim \
    -t hera:latest \
    .
}

build_full() {
  echo ">>> building hera:full (${here}/deployments/Dockerfile.full)"
  time docker build \
    -f deployments/Dockerfile.full \
    -t hera:full \
    .
}

case "$VARIANT" in
  slim) build_slim ;;
  full) build_full ;;
  both) build_slim; build_full ;;
  *)
    echo "unknown variant: $VARIANT (expected slim | full | both)" >&2
    exit 2
    ;;
esac

echo ""
echo "Build complete. Image sizes:"
docker image ls --filter reference='hera:*' --format 'table {{.Repository}}:{{.Tag}}\t{{.Size}}'
