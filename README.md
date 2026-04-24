# Hera

A multi-platform AI agent built in Go.

## Features

- **12 LLM Providers** — OpenAI, Anthropic, Gemini, Mistral, OpenRouter, Ollama, Nous, HuggingFace, GLM, Kimi, MiniMax, Compatible
- **18 Platform Adapters** — CLI, Telegram, Discord, Slack, WhatsApp, Signal, Matrix, Email, SMS, Home Assistant, DingTalk, Feishu, WeCom, Mattermost, BlueBubbles, Webhook, API Server, MCP Bridge
- **100+ Built-in Tools** — File ops, web search, code execution, memory, vision, voice, browser automation, and more
- **Skill System** — 400+ bundled skills synced on first run, user-editable, preserved across upgrades
- **SQLite Memory** — Persistent memory with FTS5 full-text search
- **Context Compression** — Auto-summarize long conversations
- **Cron Scheduler** — Schedule tasks with natural language (`every monday at 9am`)
- **Smart Routing** — Automatically routes complex requests to stronger models
- **Injection Blocking** — Detects and blocks prompt injection attempts
- **MCP Server** — Model Context Protocol for IDE integration
- **ACP Adapter** — Agent Client Protocol for editor integration
- **Plugin System** — Install third-party plugins from git repos (`hera plugins install owner/repo`)

## Current State vs Roadmap

Features listed above are wired and reachable from the main binary in v0.13.2. Some features were defined but not wired in earlier versions and are now active.

| Feature | v0.11.0 | v0.11.1 | Notes |
|---------|---------|---------|-------|
| Smart routing | Computed, discarded | Active | Routes model per request |
| Injection blocking | Warned only | Blocks + counts | Returns refusal at `InjectionHigh` |
| Skill generator | Defined | Wired | LLM generates skill content from description |
| Cron scheduler | Stub | Real scheduler | NL cron, SQLite-persisted jobs |
| MCP AttachmentStore | nil | Wired | attachment_* tools now functional |
| MCP PermissionStore | nil | Wired | permissions_* tools now functional |
| Platform adapters | 7 | 18 | 11 new adapters wired |
| LLM providers | 7 (some binaries) | 12 (all binaries) | RegisterAll helper |
| image_generation | Defined | Registered | DALL-E 3 (requires API key) |
| neutts_synth | Defined | Registered | Voice cloning (requires NeuTTS) |
| /health endpoint | None | :8080/health | Always-on, Docker healthcheck |

### Sub-agent Binaries

Six specialized binaries ship alongside the main `hera` and `hera-agent` entrypoints:

| Binary | Status | Purpose |
|--------|--------|---------|
| `hera-batch` | Shipped (v0.12.0) | Batch processing: run a file of prompts sequentially or in parallel, write results to file. Accepts `--input`, `--output`, `--parallel` flags. |
| `hera-swe` | Shipped (v0.12.0 + v0.12.1) | Autonomous software-engineering agent: code changes with SWE-bench framing, PR generation via `gh` CLI. |
| `hera-mcp` | Shipped | Model Context Protocol server for IDE integration (stdin/stdout JSON-RPC). |
| `hera-acp` | Shipped | Agent Client Protocol HTTP server (`:9090`) for editor integration. |
| `hera-rl` | Shipped | Reinforcement learning training runner with RL-focused toolset and extended timeouts. |
| `hera-supervisor` | Shipped (v0.13.0) | Lifecycle supervisor: reads `agents.yaml`, manages named agent processes, exposes `/supervisor/status` and `/supervisor/health`. |

All binaries are fully functional. See [ROADMAP.md](ROADMAP.md) for planned enhancements.

## Quick Start

```bash
git clone https://github.com/sadewadee/hera-agent
cd hera-agent
go build -o bin/hera         ./cmd/hera
go build -o bin/hera-agent   ./cmd/hera-agent

# Seed skills + directory structure (idempotent — safe to run again after upgrade)
HERA_BUNDLED=$(pwd) ./bin/hera init

# Configure LLM provider interactively
./bin/hera setup

# Chat
./bin/hera chat
```

## Installation

### Clone + build (recommended for local use)

```bash
git clone https://github.com/sadewadee/hera-agent
cd hera-agent
go build -o bin/hera         ./cmd/hera
go build -o bin/hera-agent   ./cmd/hera-agent

HERA_BUNDLED=$(pwd) ./bin/hera init
./bin/hera setup
./bin/hera chat
```

`hera init` is idempotent — run it again after upgrading to pick up new bundled skills
without overwriting any skills you have edited.

#### Environment variables

| Variable | Default | Description |
|----------|---------|-------------|
| `HERA_HOME` | `~/.hera` | User data directory (config, db, skills, plugins) |
| `HERA_BUNDLED` | `<binary-dir>/../share/hera` | Read-only bundled assets (skills, example configs) |

GoReleaser release tarballs place the bundled assets at the right relative path automatically.
For `go install` users the bundled dir is the `skills/` and `configs/` directories from the
repository — clone the repo and set `HERA_BUNDLED` to point at it, or run `hera init` after
placing them at `<binary-dir>/../share/hera/`.

### Docker

Hera ships two image variants:

| Variant | Base | Size | Use for |
|---------|------|------|---------|
| **slim** (default) | `alpine:3.19` + Go binary | ~30 MB | Chatbot-only deployments. No Python runtime. |
| **full** | `python:3.12-slim` + Go binary + `hera-python-mcp` | ~180 MB | Self-improvement / multilingual: the agent can call `python_exec`, register persistent Python tools, and `pip install` into an isolated venv. |

Images are published to both **Docker Hub** and **GitHub Container Registry (GHCR)**
on every release tag. Pull from whichever registry you prefer:

```bash
# Docker Hub
docker pull sadewadee/hera:latest
docker pull sadewadee/hera:v0.12.0

# GitHub Container Registry (alternative)
docker pull ghcr.io/sadewadee/hera:latest
docker pull ghcr.io/sadewadee/hera:slim
docker pull ghcr.io/sadewadee/hera:v0.12.0
```

Start with docker compose:

```bash
# Slim (default):
docker compose -f deployments/docker-compose.yml up -d

# Full (Python MCP enabled):
docker compose -f deployments/docker-compose.full.yml up -d
```

The container entrypoint runs `hera init --ensure` before starting `hera-agent`,
so bundled skills are seeded into the `/data` volume on first boot and upgraded
in place on subsequent boots (user-edited skills are never overwritten).

Mount `/data` as a named or host volume to persist config, memory, and skill edits:

```yaml
volumes:
  - hera-data:/data
```

**Verify skills loaded:**
```bash
docker exec hera-agent sh -c 'ls $HERA_HOME/skills | wc -l'
# Expected: >400 skill directories/files
```

To enable the Python MCP server when running the full image, add to config:

```yaml
mcp_servers:
  python:
    command: python3
    args: ["-m", "hera_python_mcp"]
    enabled: true
```

### From Source

```bash
git clone https://github.com/sadewadee/hera.git
cd hera
go build -o bin/hera ./cmd/hera
go build -o bin/hera-agent ./cmd/hera-agent
HERA_BUNDLED=$(pwd) hera init
hera setup
```

## Configuration

`hera init` seeds `$HERA_HOME/config.yaml` from the bundled example config if it does not
already exist. You can also copy it manually:

```bash
cp configs/hera.example.yaml ~/.hera/config.yaml
```

Key settings:

```yaml
agent:
  default_provider: openai
  default_model: gpt-4o

providers:
  openai:
    api_key: ${OPENAI_API_KEY}

memory:
  provider: sqlite   # or: mem0, hindsight, holographic, honcho, byterover, openviking, retaindb, supermemory
```

### Memory Providers

| Provider | Type | Requires |
|----------|------|---------|
| `sqlite` (default) | Local SQLite + FTS5 | Nothing — built-in |
| `holographic` | Local SQLite graph | Nothing — built-in, no external API |
| `mem0` | Cloud semantic memory | `MEM0_API_KEY` |
| `hindsight` | Cloud recall | `HINDSIGHT_API_KEY` |
| `honcho` | Cloud peer cards | `HONCHO_API_KEY` |
| `byterover` | Cloud RAG | `BYTEROVER_API_KEY` |
| `openviking` | Cloud vector search | `OPENVIKING_API_KEY` |
| `retaindb` | Cloud database | `RETAINDB_API_KEY` |
| `supermemory` | Cloud memory | `SUPERMEMORY_API_KEY` |

Set `memory.provider: <name>` in your config and the matching env var to activate.

## Platform Support

| Platform | Status | Required env |
|----------|--------|--------------|
| CLI | Wired + smoke-tested | — |
| Telegram | Wired + smoke-tested | `TELEGRAM_BOT_TOKEN` |
| Discord | Wired + smoke-tested | `DISCORD_BOT_TOKEN` |
| Slack | Wired + smoke-tested | `SLACK_BOT_TOKEN`, `SLACK_APP_TOKEN` |
| Webhook | Wired + smoke-tested | — |
| API Server | Wired + smoke-tested | — |
| Threads | Wired + smoke-tested | `THREADS_ACCESS_TOKEN` |
| WhatsApp | Wired (v0.11.1, not integration-tested) | `WHATSAPP_ACCESS_TOKEN`, `WHATSAPP_PHONE_NUMBER_ID` |
| Signal | Wired (v0.11.1, not integration-tested) | `SIGNAL_SERVICE_URL`, `SIGNAL_PHONE_NUMBER` |
| Matrix | Wired (v0.11.1, not integration-tested) | `MATRIX_ACCESS_TOKEN`, `MATRIX_HOMESERVER_URL` |
| Email | Wired (v0.11.1, not integration-tested) | `SMTP_HOST`, `SMTP_USER`, `SMTP_PASSWORD` |
| SMS (Twilio) | Wired (v0.11.1, not integration-tested) | `TWILIO_ACCOUNT_SID`, `TWILIO_AUTH_TOKEN` |
| Home Assistant | Wired (v0.11.1, not integration-tested) | `HA_LONG_LIVED_TOKEN`, `HA_URL` |
| DingTalk | Wired (v0.11.1, not integration-tested) | `DINGTALK_ACCESS_TOKEN` |
| Feishu | Wired (v0.11.1, not integration-tested) | `FEISHU_APP_ID`, `FEISHU_APP_SECRET` |
| WeCom | Wired (v0.11.1, not integration-tested) | `WECOM_CORP_ID`, `WECOM_CORP_SECRET` |
| Mattermost | Wired (v0.11.1, not integration-tested) | `MATTERMOST_TOKEN`, `MATTERMOST_SERVER_URL` |
| BlueBubbles | Wired (v0.11.1, not integration-tested) | `BLUEBUBBLES_API_URL`, `BLUEBUBBLES_PASSWORD` |
| MCP Bridge | Wired + smoke-tested | — |

## Plugin System

```bash
# Install a plugin from GitHub (git clone model)
hera plugins install owner/repo

# List installed plugins
hera plugins list

# Enable / disable
hera plugins enable owner/repo
hera plugins disable owner/repo

# Remove
hera plugins remove owner/repo
```

Plugins can contribute skills, hooks, custom tools, and MCP server definitions.
Installed to `$HERA_HOME/plugins/<owner-repo>/`.

## Architecture

```
cmd/
  hera/         CLI entry point
  hera-agent/   Headless agent (daemon)
  hera-mcp/     MCP server
  hera-acp/     ACP adapter
  hera-batch/   Batch processor (v0.12.x)
  hera-swe/     SWE self-improvement agent (v0.12.x)

internal/
  agent/        Core agent logic
  llm/          LLM providers
  memory/       SQLite + FTS5 memory
  tools/        Tool system + 40+ built-in tools
  skills/       Skill loader
  gateway/      Multi-platform gateway
  cli/          CLI interface
  mcp/          MCP server
  acp/          ACP adapter
  cron/         Scheduled tasks
  environments/ RL environments + benchmarks
  config/       Configuration
  paths/        Filesystem path resolution (HERA_HOME, HERA_BUNDLED)
  syncer/       Bundled skills sync with manifest-based copy-on-modify
  plugins/      Plugin system (git clone model)
```

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

## License

MIT
