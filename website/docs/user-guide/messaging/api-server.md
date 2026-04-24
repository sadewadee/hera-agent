---
title: API Server
sidebar_label: API Server
---

# API Server

The API server exposes Hera as an HTTP REST service, suitable for integration with custom frontends, mobile apps, or backend services.

## Configuration

### Environment Variables

```bash
API_SERVER_LISTEN_ADDR=:8080
API_SERVER_AUTH_TOKEN=your-bearer-token   # Required for authentication
```

### Config File (`~/.hera/config.yaml`)

```yaml
gateway:
  platforms:
    apiserver:
      enabled: true
      extra:
        listen_addr: ":8080"
        auth_token: ${API_SERVER_AUTH_TOKEN}
```

## Starting the API Server

```bash
hera gateway --platform apiserver
```

Or as a standalone binary:

```bash
hera-agent --listen :8080
```

## Authentication

All requests require a Bearer token in the `Authorization` header:

```
Authorization: Bearer your-bearer-token
```

## Endpoints

### POST /chat

Send a message and receive a response.

**Request:**

```json
{
  "session_id": "optional-session-id",
  "message": "Explain what Docker is",
  "stream": false
}
```

**Response:**

```json
{
  "session_id": "sess_abc123",
  "response": "Docker is a platform for containerizing applications...",
  "tokens_used": 245
}
```

### POST /chat/stream

Stream a response using Server-Sent Events (SSE).

```bash
curl -X POST http://localhost:8080/chat/stream \
  -H "Authorization: Bearer your-token" \
  -H "Content-Type: application/json" \
  -d '{"message": "Write a haiku about Go"}' \
  --no-buffer
```

### GET /health

Returns `200 OK` with `{"status": "ok"}` when the server is running.

### GET /tools

Lists all registered tools.

### GET /sessions/{id}

Retrieves the message history for a session.

### DELETE /sessions/{id}

Clears a session's conversation history.

## Session Management

- If `session_id` is omitted, Hera generates one and returns it
- Sessions expire after `gateway.session_timeout` minutes of inactivity (default: 30)
- Each session maintains independent conversation context

## Production Setup

For production deployment:

1. Put Hera behind a reverse proxy (Caddy, nginx) for TLS termination
2. Set a strong, random `API_SERVER_AUTH_TOKEN`
3. Use Docker or systemd for process management

Example Caddy config:

```
hera.yourdomain.com {
  reverse_proxy localhost:8080
}
```

See the [Docker deployment guide](/docs/getting-started/installation#docker) for containerized setups.
