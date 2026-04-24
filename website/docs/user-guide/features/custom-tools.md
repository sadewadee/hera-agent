# Custom Tools

You can define your own tools in `config.yaml` without writing Go code. Custom tools execute shell commands or make HTTP calls and return the output to the agent.

## Tool types

| Type | Runs | Use when |
|------|------|---------|
| `command` | Shell command | Wrapping CLI tools, scripts |
| `http` | HTTP request | Calling APIs |
| `script` | Inline script | Multi-step logic |

## Command tools

A command tool runs a shell command and returns stdout:

```yaml
tools:
  - name: run_tests
    description: "Run the Go test suite and return results"
    type: command
    command: "go test ./... -v"

  - name: git_status
    description: "Show the current git status of the repository"
    type: command
    command: "git status --short"

  - name: docker_ps
    description: "List running Docker containers"
    type: command
    command: "docker ps --format 'table {{.Names}}\t{{.Status}}\t{{.Ports}}'"
```

## Command tools with parameters

Use `{param_name}` placeholders in the command string:

```yaml
tools:
  - name: deploy
    description: "Deploy to a named environment"
    type: command
    command: "./scripts/deploy.sh {environment}"
    parameters:
      - name: environment
        type: string
        description: "Target environment: staging or production"
        required: true
    timeout: 120  # seconds

  - name: search_logs
    description: "Search application logs for a pattern"
    type: command
    command: "grep -n '{pattern}' /var/log/app.log | tail -50"
    parameters:
      - name: pattern
        type: string
        description: "Search pattern (regex supported)"
        required: true
```

## HTTP tools

HTTP tools call a URL and return the response body:

```yaml
tools:
  - name: check_service_health
    description: "Check if the API service is healthy"
    type: http
    url: "https://api.myapp.com/health"
    method: GET
    headers:
      Authorization: "Bearer ${API_TOKEN}"

  - name: create_ticket
    description: "Create a support ticket"
    type: http
    url: "https://api.myapp.com/tickets"
    method: POST
    headers:
      Content-Type: "application/json"
      Authorization: "Bearer ${API_TOKEN}"
    parameters:
      - name: title
        type: string
        description: "Ticket title"
        required: true
      - name: description
        type: string
        description: "Ticket description"
        required: true
```

## Parameter types

| Type | JSON type | Example |
|------|-----------|---------|
| `string` | `"string"` | `"production"` |
| `integer` | `"integer"` | `42` |
| `boolean` | `"boolean"` | `true` |

## Full CustomToolConfig reference

```yaml
tools:
  - name: string           # Required: unique tool name
    description: string    # Required: shown to LLM to decide when to use
    type: string           # Required: "command", "http", or "script"
    command: string        # For type=command: shell command to run
    url: string            # For type=http: target URL
    method: string         # For type=http: GET, POST, PUT, DELETE
    headers:               # For type=http: request headers
      Key: Value
    parameters:            # Tool parameters
      - name: string       # Parameter name
        type: string       # "string", "integer", "boolean"
        description: string
        required: bool
    timeout: int           # Timeout in seconds (default: 30)
```

## Using custom tools

Once defined, the agent uses custom tools like built-in tools:

```
You: Run the tests

Hera: [using run_tests]
ok      github.com/sadewadee/hera/internal/agent    0.234s
ok      github.com/sadewadee/hera/internal/memory   0.089s
FAIL    github.com/sadewadee/hera/internal/tools    0.156s

There are test failures in internal/tools. Would you like me to look at the details?
```

## Security note

Command tools run with the same permissions as the Hera process. Be careful with commands that accept user-provided parameters — avoid shell injection by validating inputs.
