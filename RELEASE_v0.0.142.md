# Release v0.0.142 — Initial Public Release

**Released:** 2026-04-24

This is the first public release of Hera Agent, a self-improving, multi-platform
AI agent built in Go.

## Versioning

Public releases follow the `0.0.X` scheme where `X` maps to the internal
development version. This release (`0.0.142`) corresponds to internal
`v0.14.2`.

Subsequent public releases will increment `X` as development milestones ship.

## What's Included

- Eight binaries for different deployment modes: `hera` (interactive CLI),
  `hera-agent` (daemon + HTTP health), `hera-mcp` (MCP stdio server),
  `hera-acp` (ACP HTTP), `hera-batch`, `hera-swe`, `hera-rl`,
  `hera-supervisor`.
- Twelve LLM providers via raw `net/http` + SSE (no vendor SDKs): OpenAI,
  Anthropic, Gemini, Ollama, Mistral, OpenRouter, Nous, HuggingFace, GLM,
  Kimi, MiniMax, plus OpenAI-compatible generic.
- Eighteen platform adapters: CLI, Telegram, Discord, Slack, WhatsApp,
  Signal, Matrix, Email, SMS, Home Assistant, DingTalk, Feishu, WeCom,
  Mattermost, BlueBubbles, Webhook, API server, MCP.
- Persistent SQLite + FTS5 memory with pluggable providers (sqlite,
  holographic, mem0, hindsight, honcho, byterover, openviking, retaindb,
  supermemory).
- Pluggable context engines (`agent.compression.engine: <name>` in
  `hera.yaml`) — ships with a built-in compressor; interface is open for
  third-party engines.
- 70+ built-in tools (file I/O, shell, web search, archives, CSV/PDF,
  database, image gen, voice synth, session search, more).
- Skill library (405+ Markdown skills with YAML frontmatter) with
  per-platform filtering, first-run sync via `hera init`, and copy-on-modify
  upgrades.
- Multi-agent orchestration: `delegate_task` tool, shared event bus,
  supervisor with Spawn/Stop/Restart lifecycle, per-agent token/cost budgets.
- Cron scheduler (hand-written 5-field parser) with natural-language input,
  SQLite-persisted jobs.
- HERA_HOME-centric path handling with `~` expansion, `$HERA_HOME` and
  `.hera/...` auto-resolution.

## Quick Start

```bash
git clone https://github.com/sadewadee/hera-agent
cd hera-agent
go build -o bin/hera ./cmd/hera
./bin/hera init            # seed $HERA_HOME with skills, configs, SOUL.md
./bin/hera setup           # interactive LLM API-key wizard
./bin/hera chat            # start an interactive session
```

Or via Docker:

```bash
docker compose -f deployments/docker-compose.yml up -d
```

## Verification

```bash
go build ./...             # all 8 binaries build
go test -race -count=1 ./... # test suite passes
./bin/hera version         # reports 0.0.142
```

## License

See `LICENSE`.
