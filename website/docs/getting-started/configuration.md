# Configuration

Hera is configured via a YAML file at `~/.hera/config.yaml`. You can also use environment variables (prefixed `HERA_`) for any config key, or a `.env` file in `~/.hera/` or the current directory.

## Config file location

| Path | Purpose |
|------|---------|
| `~/.hera/config.yaml` | Primary config (created by `hera setup`) |
| `~/.hera/.env` | Secrets / API keys (not committed to git) |
| `./.env` | Project-local overrides |

## Minimal configuration

The only required config is an LLM provider API key. You can set it as an environment variable:

```bash
export OPENAI_API_KEY=sk-...
hera chat  # works with zero config file
```

## Full config reference

```yaml
# ~/.hera/config.yaml

agent:
  default_provider: openai       # which provider to use by default
  default_model: gpt-4o          # which model to use by default
  personality: helpful           # built-in personality name or "custom"
  soul_file: ~/.hera/SOUL.md     # path to custom SOUL.md personality file
  max_tool_calls: 20             # max tool calls per agent turn
  memory_nudge_interval: 10      # remind agent of memories every N turns
  skill_nudge_interval: 15       # remind agent of available skills every N turns
  smart_routing: true            # route simple queries to cheaper models
  prompt_caching: false          # enable prompt caching (Anthropic only)

  compression:
    enabled: true                # auto-summarize long contexts
    threshold: 0.5               # compress when context is 50% full
    target_ratio: 0.2            # compress down to 20% of window
    protected_turns: 5           # always keep the last 5 turns verbatim
    summary_model: ""            # model used for summaries (defaults to default_model)

# LLM Providers
providers:
  openai:
    type: openai
    api_key: ${OPENAI_API_KEY}   # ${VAR} syntax reads from environment
    models:
      - gpt-4o
      - gpt-4o-mini

  anthropic:
    type: anthropic
    api_key: ${ANTHROPIC_API_KEY}
    models:
      - claude-sonnet-4-20250514
      - claude-3-5-haiku-20241022

  gemini:
    type: gemini
    api_key: ${GEMINI_API_KEY}
    models:
      - gemini-2.0-flash

  mistral:
    type: mistral
    api_key: ${MISTRAL_API_KEY}
    models:
      - mistral-large-latest

  openrouter:
    type: openrouter
    api_key: ${OPENROUTER_API_KEY}
    models:
      - openai/gpt-4o
      - anthropic/claude-sonnet-4-20250514

  # Local / self-hosted OpenAI-compatible endpoints (e.g. Ollama, LM Studio)
  local:
    type: compatible
    base_url: http://localhost:11434/v1
    models:
      - llama3.2
      - codellama

# Memory System
memory:
  provider: sqlite               # only "sqlite" supported currently
  db_path: ~/.hera/hera.db      # SQLite database file
  max_results: 10               # max memories returned per query

# Multi-Platform Messaging Gateway
gateway:
  session_timeout: 30           # session expiry in minutes
  human_delay: true             # simulate human typing speed
  delay_ms_per_char: 30         # milliseconds per character when human_delay=true
  allow_all: false              # if true, accept messages from any user (use carefully)

  platforms:
    telegram:
      enabled: true
      token: ${TELEGRAM_BOT_TOKEN}
      allow_list:               # Telegram user IDs allowed to chat
        - "123456789"

    discord:
      enabled: true
      token: ${DISCORD_BOT_TOKEN}

    slack:
      enabled: true
      token: ${SLACK_BOT_TOKEN}

    whatsapp:
      enabled: true
      token: ${WHATSAPP_TOKEN}
      extra:
        phone_number_id: "..."

# CLI Settings
cli:
  skin: default                 # UI skin: "default", "minimal", "rich"
  profile: default              # named profile to load

# Cron / Scheduled Tasks
cron:
  enabled: false                # set true to enable scheduled jobs

# Security
security:
  redact_pii: false             # redact emails, phone numbers, SSNs from messages
  dangerous_approve: true       # auto-approve dangerous tool calls (careful!)
  protected_paths:              # paths the agent cannot modify
    - ~/.ssh
    - ~/.gnupg
    - ~/.aws/credentials

# Custom Tools (YAML-defined)
tools:
  - name: run_tests
    description: "Run the project test suite"
    type: command
    command: "go test ./..."

# Hooks
hooks:
  - name: log_messages
    event: after_message
    type: command
    command: "echo 'Message processed'"
    async: true

# MCP Servers
mcp_servers:
  - name: filesystem
    command: npx
    args: ["-y", "@modelcontextprotocol/server-filesystem", "/tmp"]
```

## Environment variables

Every config key can be set via `HERA_` prefixed environment variables. Nested keys use underscores:

| Env Var | Config Key | Example |
|---------|-----------|---------|
| `HERA_AGENT_DEFAULT_PROVIDER` | `agent.default_provider` | `anthropic` |
| `HERA_AGENT_DEFAULT_MODEL` | `agent.default_model` | `claude-sonnet-4-20250514` |
| `HERA_AGENT_SMART_ROUTING` | `agent.smart_routing` | `true` |
| `HERA_SECURITY_REDACT_PII` | `security.redact_pii` | `true` |
| `HERA_MEMORY_DB_PATH` | `memory.db_path` | `/data/hera.db` |

Provider API keys can be set directly without the `HERA_` prefix:

| Provider | Environment Variable |
|----------|---------------------|
| OpenAI | `OPENAI_API_KEY` |
| Anthropic | `ANTHROPIC_API_KEY` |
| Gemini | `GEMINI_API_KEY` |
| Mistral | `MISTRAL_API_KEY` |
| OpenRouter | `OPENROUTER_API_KEY` |
| Nous | `NOUS_API_KEY` |
| HuggingFace | `HF_TOKEN` |
| GLM | `GLM_API_KEY` |
| Kimi (Moonshot) | `MOONSHOT_API_KEY` |
| MiniMax | `MINIMAX_API_KEY` |
| Custom compatible | `COMPATIBLE_API_KEY` |

## Personalities

Hera ships with several built-in personalities. Set `agent.personality` to one of:

- `helpful` — balanced, general-purpose assistant (default)
- `coder` — focused on software development
- `researcher` — analytical, cites sources
- `concise` — brief responses

For a fully custom personality, create `~/.hera/SOUL.md` and set `agent.soul_file` to point to it. The content of `SOUL.md` is injected into the system prompt.

Example `SOUL.md`:

```markdown
You are a senior Go engineer. You prefer standard library solutions over
third-party dependencies. You always check for race conditions and write
table-driven tests with testify.
```
