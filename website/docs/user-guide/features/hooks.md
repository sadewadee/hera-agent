# Hooks

Hooks let you run custom commands or HTTP calls at specific points in the agent's lifecycle. Use hooks for logging, notifications, integrations, or side effects.

## Hook events

| Event | When it fires |
|-------|--------------|
| `before_message` | Before the agent processes a user message |
| `after_message` | After the agent produces a response |
| `before_tool` | Before any tool is called |
| `after_tool` | After a tool call completes |
| `on_error` | When an error occurs in the agent loop |

## Configuration

```yaml
hooks:
  - name: log_messages
    event: after_message
    type: command
    command: "echo 'Response generated' >> ~/.hera/agent.log"
    async: true

  - name: notify_slack
    event: on_error
    type: http
    url: "https://hooks.slack.com/services/..."
    async: true

  - name: audit_tools
    event: before_tool
    type: command
    command: "logger 'hera tool call'"
    async: false  # blocks until hook completes
```

## HookConfig fields

| Field | Type | Description |
|-------|------|-------------|
| `name` | string | Identifier for this hook |
| `event` | string | Lifecycle event to listen for |
| `type` | string | `"command"` or `"http"` |
| `command` | string | Shell command (for type=command) |
| `url` | string | URL to POST to (for type=http) |
| `async` | bool | Run hook without blocking the agent (default: false) |

## Async vs synchronous hooks

- **`async: true`** — hook fires in a goroutine; agent does not wait for it. Use for logging, notifications.
- **`async: false`** — agent waits for the hook to complete before continuing. Use when the hook result matters.

## HTTP hooks

HTTP hooks send a POST request to the configured URL with a JSON body describing the event:

```json
{
  "event": "after_message",
  "platform": "telegram",
  "user_id": "123456789",
  "timestamp": "2026-04-14T10:00:00Z"
}
```

## Example: Webhook notification on response

```yaml
hooks:
  - name: webhook_after_response
    event: after_message
    type: http
    url: "https://my-app.com/webhooks/hera"
    async: true
```

## Example: Write an audit log before tool calls

```yaml
hooks:
  - name: tool_audit_log
    event: before_tool
    type: command
    command: "date >> ~/.hera/tool-audit.log"
    async: true
```

## Example: Send desktop notification

```yaml
hooks:
  - name: desktop_notify
    event: after_message
    type: command
    command: "osascript -e 'display notification \"Hera responded\" with title \"Hera\"'"
    async: true
```

## Built-in hooks

Some platform adapters register built-in hooks for their own needs (e.g., the gateway's sticker/delivery receipt handling). These run automatically and are not configurable.
