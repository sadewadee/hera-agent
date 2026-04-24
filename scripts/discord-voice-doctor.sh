#!/bin/bash
# ============================================================================
# Discord Voice Doctor -- diagnostic tool for voice channel support.
#
# Checks all dependencies, configuration, and bot permissions needed
# for Discord voice mode to work correctly.
#
# Usage:
#   bash scripts/discord-voice-doctor.sh
# ============================================================================

set -euo pipefail

OK="\033[92m[OK]\033[0m"
FAIL="\033[91m[FAIL]\033[0m"
WARN="\033[93m[!]\033[0m"

HERA_HOME="${HERA_HOME:-$HOME/.hera}"
ENV_FILE="$HERA_HOME/.env"

# Load env file if it exists
if [ -f "$ENV_FILE" ]; then
    set -a
    # shellcheck disable=SC1090
    source "$ENV_FILE"
    set +a
fi

mask() {
    local val="$1"
    if [ ${#val} -lt 8 ]; then
        echo "****"
    else
        echo "${val:0:4}$(printf '*%.0s' $(seq 1 $((${#val} - 4))))"
    fi
}

check() {
    local label="$1"
    local ok="$2"
    local detail="${3:-}"
    if [ "$ok" = "true" ]; then
        printf "  %b %s" "$OK" "$label"
    else
        printf "  %b %s" "$FAIL" "$label"
    fi
    if [ -n "$detail" ]; then
        printf "  (%s)" "$detail"
    fi
    echo ""
    [ "$ok" = "true" ]
}

warn() {
    local label="$1"
    local detail="${2:-}"
    printf "  %b %s" "$WARN" "$label"
    if [ -n "$detail" ]; then
        printf "  (%s)" "$detail"
    fi
    echo ""
}

section() {
    echo ""
    echo -e "\033[1m$1\033[0m"
}

# ============================================================================
# 1. System Dependencies
# ============================================================================
section "1. System Dependencies"

check "ffmpeg installed" "$(command -v ffmpeg &>/dev/null && echo true || echo false)" \
    "$(ffmpeg -version 2>/dev/null | head -1 || echo 'not found')" || true

check "opus codec available" "$(ffmpeg -codecs 2>/dev/null | grep -q opus && echo true || echo false)" || true

check "node.js installed" "$(command -v node &>/dev/null && echo true || echo false)" \
    "$(node --version 2>/dev/null || echo 'not found')" || true

# ============================================================================
# 2. Discord Configuration
# ============================================================================
section "2. Discord Configuration"

DISCORD_TOKEN="${DISCORD_BOT_TOKEN:-}"
if [ -n "$DISCORD_TOKEN" ]; then
    check "DISCORD_BOT_TOKEN set" "true" "$(mask "$DISCORD_TOKEN")"
else
    check "DISCORD_BOT_TOKEN set" "false" "Not found in environment or $ENV_FILE"
fi

# ============================================================================
# 3. Audio Pipeline
# ============================================================================
section "3. Audio Pipeline"

# Check for common TTS/STT env vars
OPENAI_KEY="${OPENAI_API_KEY:-}"
if [ -n "$OPENAI_KEY" ]; then
    check "OPENAI_API_KEY set (for TTS/STT)" "true" "$(mask "$OPENAI_KEY")"
else
    warn "OPENAI_API_KEY not set" "Needed for OpenAI TTS/STT; other providers may work"
fi

DEEPGRAM_KEY="${DEEPGRAM_API_KEY:-}"
if [ -n "$DEEPGRAM_KEY" ]; then
    check "DEEPGRAM_API_KEY set (alt STT)" "true" "$(mask "$DEEPGRAM_KEY")"
else
    warn "DEEPGRAM_API_KEY not set" "Optional: alternative STT provider"
fi

# ============================================================================
# 4. Network Connectivity
# ============================================================================
section "4. Network Connectivity"

check "Discord API reachable" \
    "$(curl -s -o /dev/null -w '%{http_code}' https://discord.com/api/v10/gateway 2>/dev/null | grep -q '200\|401' && echo true || echo false)" || true

check "Discord Gateway (WSS)" \
    "$(curl -s -o /dev/null -w '%{http_code}' https://gateway.discord.gg/ 2>/dev/null | grep -q '200\|301\|101' && echo true || echo false)" || true

# ============================================================================
# Summary
# ============================================================================
section "Summary"
echo "  If any checks failed, resolve them before enabling Discord voice mode."
echo "  See: https://github.com/sadewadee/hera/blob/main/docs/discord-voice.md"
echo ""
