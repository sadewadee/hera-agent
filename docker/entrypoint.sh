#!/usr/bin/env bash
set -euo pipefail

# Hera Docker entrypoint
echo "Starting Hera v0.1.0..."

# Load config from environment
export HERA_CONFIG_PATH="${HERA_CONFIG_PATH:-/app/config/hera.yaml}"
export HERA_DATA_DIR="${HERA_DATA_DIR:-/data}"

# Create data directory if needed
mkdir -p "$HERA_DATA_DIR"

# Handle signals for graceful shutdown
trap 'echo "Shutting down..."; kill -TERM "$PID"; wait "$PID"' SIGTERM SIGINT

# Run hera
exec /app/hera "$@" &
PID=$!
wait "$PID"
