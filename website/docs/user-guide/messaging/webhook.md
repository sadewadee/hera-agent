---
title: Generic Webhook
sidebar_label: Webhook
---

# Generic Webhook Integration

The webhook adapter lets any service send messages to Hera over HTTP. Send a JSON payload and receive a response — no platform-specific SDK required.

## How It Works

Hera listens on a configurable address and port. Any HTTP client that can POST JSON can send a message and receive a response.

## Configuration

### Environment Variables

```bash
WEBHOOK_LISTEN_ADDR=:8090
WEBHOOK_SECRET=your-shared-secret    # Optional: validates X-Webhook-Secret header
WEBHOOK_CALLBACK_URL=                # Optional: URL to POST responses to
```

### Config File (`~/.hera/config.yaml`)

```yaml
gateway:
  platforms:
    webhook:
      enabled: true
      extra:
        listen_addr: ":8090"
        secret: ${WEBHOOK_SECRET}
        callback_url: https://your-service.com/hera-callback
```

## Starting Hera with Webhook

```bash
hera gateway --platform webhook
```

## Sending a Message

```bash
curl -X POST http://localhost:8090/webhook \
  -H "Content-Type: application/json" \
  -H "X-Webhook-Secret: your-shared-secret" \
  -d '{
    "session_id": "user-123",
    "message": "What is the weather like today?"
  }'
```

## Response Format

Hera responds synchronously:

```json
{
  "session_id": "user-123",
  "response": "I don't have real-time weather data, but I can help you check...",
  "tool_calls": []
}
```

Or, if a `callback_url` is configured, Hera POSTs the response there asynchronously and returns `202 Accepted` immediately.

## Request Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `session_id` | string | Yes | Identifies the conversation session |
| `message` | string | Yes | The user's message |
| `user_id` | string | No | Optional user identifier for allow-list checks |

## Security

- Set `WEBHOOK_SECRET` to require a shared secret header
- Validate the secret in your sending service
- Use HTTPS in production (put Hera behind a reverse proxy like Caddy or nginx)
- The `X-Webhook-Secret` header value must match `WEBHOOK_SECRET` exactly
