# Hera WhatsApp Sidecar

Node.js sidecar that connects to WhatsApp via Baileys (multi-device) and communicates with the Hera Go process over a Unix socket.

## Setup

```bash
cd sidecar/whatsapp
npm install
```

## Running

```bash
npm start
```

On first run, a QR code will be displayed in the terminal. Scan it with WhatsApp to authenticate.

## How It Works

The sidecar connects to WhatsApp using the Baileys library and opens a Unix socket for IPC with the Hera Go process. Messages are exchanged as newline-delimited JSON.

### Protocol

**Go to Sidecar (commands):**
```json
{"type":"send","to":"1234567890@s.whatsapp.net","text":"Hello"}
{"type":"status"}
```

**Sidecar to Go (events):**
```json
{"type":"message","from":"1234567890@s.whatsapp.net","text":"Hi","timestamp":1234567890}
{"type":"qr","data":"qr-code-string"}
{"type":"connected","jid":"xxx@s.whatsapp.net"}
{"type":"disconnected","reason":"..."}
```

## Configuration

Environment variables:

| Variable | Default | Description |
|----------|---------|-------------|
| `HERA_WA_SOCKET_PATH` | `/tmp/hera-whatsapp.sock` | Unix socket path |
| `HERA_WA_AUTH_DIR` | `~/.config/hera/whatsapp-auth` | Auth state directory |

## Auth State

Authentication state is persisted in the auth directory. Delete this directory to force re-authentication.
