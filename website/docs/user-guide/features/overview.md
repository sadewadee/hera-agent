# Features Overview

Hera is a multi-platform AI agent with a broad set of built-in capabilities. This page lists every major feature and where to find detailed documentation.

## Core Agent

| Feature | Description | Docs |
|---------|-------------|------|
| Tool calling | Agent uses 40+ built-in tools to act on your behalf | [Tool Calling](./tool-calling.md) |
| Memory | Persistent SQLite memory with full-text search | [Memory](./memory.md) |
| Skills | 170+ Markdown instruction files that extend agent behavior | [Skills](./skills.md) |
| Context compression | Auto-summarize long conversations to stay within context windows | [Context Compression](./context-compression.md) |
| Smart routing | Route simple queries to cheaper models automatically | [Smart Routing](./smart-routing.md) |
| Cron / Scheduled tasks | Run agent tasks on a schedule | [Cron](./cron.md) |

## Security

| Feature | Description | Docs |
|---------|-------------|------|
| PII redaction | Strip emails, phone numbers, SSNs, credit cards before sending to LLM | [PII Redaction](./pii-redaction.md) |
| Prompt injection detection | Detect and block jailbreak / injection attempts | [Prompt Injection](./prompt-injection.md) |
| Protected paths | Prevent the agent from modifying sensitive files | [Configuration](../../getting-started/configuration.md) |

## LLM Providers

Hera connects to 15+ LLM providers via raw REST APIs (no vendor SDK required):

| Provider | Type | Notes |
|----------|------|-------|
| OpenAI | Cloud | GPT-4o, GPT-4o-mini, GPT-4-turbo |
| Anthropic | Cloud | Claude Sonnet, Haiku, Opus |
| Google Gemini | Cloud | Gemini 2.0 Flash, 2.0 Pro, 1.5 Pro |
| Mistral | Cloud | Mistral Large, Medium |
| OpenRouter | Aggregator | Access 100+ models from one key |
| Nous Research | Cloud | Hermes series |
| HuggingFace | Cloud | Inference API |
| GLM | Cloud | Zhipu GLM-4 |
| Kimi (Moonshot) | Cloud | Moonshot-v1 series |
| MiniMax | Cloud | MiniMax series |
| Ollama | Local | Any locally-running model |
| Compatible | Local/Self-hosted | Any OpenAI-compatible endpoint |

## Platform Adapters (Gateway)

Hera's gateway forwards messages from 18 platforms to the agent and returns responses:

| Platform | Status |
|----------|--------|
| CLI (interactive) | Stable |
| API Server (REST) | Stable |
| Telegram | Stable |
| Discord | Stable |
| Slack | Stable |
| WhatsApp (Cloud API) | Stable |
| Signal | Stable |
| Matrix | Stable |
| Email (IMAP/SMTP) | Stable |
| SMS (Twilio) | Stable |
| Home Assistant | Stable |
| DingTalk | Stable |
| Feishu | Stable |
| WeCom | Stable |
| Mattermost | Stable |
| BlueBubbles (iMessage) | Stable |
| Webhook | Stable |
| MCP bridge | Stable |

## Tool Categories

Built-in tools are organized by category:

| Category | Examples |
|----------|---------|
| File operations | read_file, write_file, patch, archive, csv_tool |
| Terminal | shell, terminal, process |
| Web | web_search, web_extract, http_client |
| Memory | memory (store/retrieve/search) |
| Code execution | execute_code |
| Vision | vision (analyze images) |
| Database | database_tool |
| Git | git_tool |
| Docker | docker_tool |
| Kubernetes | k8s_tool |
| SSH | ssh_tool |
| Scheduling | cronjob |
| Delegation | delegate (spawn sub-agents) |
| Notifications | notifications |
| Skills management | skills (install/list/run) |
| MCP | mcp_tool (call any MCP tool) |

## Integration Protocols

| Protocol | Purpose | Binary |
|----------|---------|--------|
| MCP (Model Context Protocol) | IDE/editor integration via stdio | `hera-mcp` |
| ACP (Agent Client Protocol) | Editor adapter with streaming | `hera-acp` |
| REST API | HTTP access for custom integrations | `hera-agent` + apiserver |

## Custom Extensions

- **[Custom Tools](./custom-tools.md)** — define tools as shell commands or HTTP calls in `config.yaml`
- **[Hooks](./hooks.md)** — run commands at agent lifecycle events
- **[MCP Servers](./mcp-servers.md)** — connect any MCP-compatible server
- **[Personalities](./personalities.md)** — define agent persona via `SOUL.md`
- **[Skills](./skills.md)** — install community skills or write your own
