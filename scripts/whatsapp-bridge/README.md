# WhatsApp Bridge

Node.js sidecar that connects Hera to WhatsApp via the
[whatsapp-web.js](https://github.com/pedroslopez/whatsapp-web.js) library.

## Quick Start

```bash
cd scripts/whatsapp-bridge
npm install
bash start.sh
```

Scan the QR code with your phone when prompted. The session persists in
`~/.hera/whatsapp-session/` so you only need to scan once.

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `HERA_HOME` | `~/.hera` | Hera data directory |
| `WHATSAPP_PORT` | `3478` | HTTP port for the bridge API |
| `WHATSAPP_ALLOWLIST` | _(none)_ | Comma-separated phone numbers to accept |

## Architecture

```
WhatsApp <--> whatsapp-web.js <--> bridge.js (HTTP API) <--> Hera Gateway
```

The bridge exposes a local HTTP API that Hera's gateway adapter calls.
Messages from WhatsApp are forwarded to Hera, and responses are sent back.

## Files

- `bridge.js` -- Main bridge server
- `allowlist.js` -- Phone number allowlist filtering
- `allowlist.test.mjs` -- Tests for allowlist module
- `package.json` -- Node.js dependencies
- `start.sh` -- Startup script

## Allowlist

If `WHATSAPP_ALLOWLIST` is set, only messages from those phone numbers
will be processed. Format: comma-separated E.164 numbers without the `+`:

```
WHATSAPP_ALLOWLIST=1234567890,0987654321
```
