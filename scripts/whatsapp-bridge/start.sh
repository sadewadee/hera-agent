#!/bin/bash
# ============================================================================
# WhatsApp Bridge -- Start script
#
# Starts the WhatsApp Web bridge (Node.js sidecar) that connects Hera
# to WhatsApp via the whatsapp-web.js library.
#
# Prerequisites:
#   - Node.js 18+
#   - npm install (in this directory)
#
# Usage:
#   cd scripts/whatsapp-bridge
#   bash start.sh
#
# Environment:
#   HERA_HOME            - Hera data directory (default: ~/.hera)
#   WHATSAPP_PORT        - Bridge HTTP port (default: 3478)
#   WHATSAPP_ALLOWLIST   - Comma-separated phone numbers (optional)
# ============================================================================

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
cd "$SCRIPT_DIR"

HERA_HOME="${HERA_HOME:-$HOME/.hera}"
WHATSAPP_PORT="${WHATSAPP_PORT:-3478}"
SESSION_DIR="$HERA_HOME/whatsapp-session"

# Ensure dependencies are installed
if [ ! -d "node_modules" ]; then
    echo "Installing dependencies..."
    npm install
fi

# Create session directory for persistent auth
mkdir -p "$SESSION_DIR"

echo "Starting WhatsApp Bridge..."
echo "  Port:     $WHATSAPP_PORT"
echo "  Session:  $SESSION_DIR"
echo ""
echo "Scan the QR code with WhatsApp when prompted."
echo "Press Ctrl+C to stop."
echo ""

export WHATSAPP_PORT
export WHATSAPP_SESSION_DIR="$SESSION_DIR"

exec node bridge.js
