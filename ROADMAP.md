# Hera Roadmap

This document tracks planned capabilities by release. Items without a version target are speculative.

## v0.11.x — Wire Ghost Features / Honest Docs

Theme of this series: features that were defined but not reachable from `main()` are wired; documentation is updated to reflect actual state.

### v0.11.1

- Platform adapter parity: 18 adapters wired (7 smoke-tested, 11 code-complete but not integration-tested)
- Smart routing: active — routes complex requests to stronger models per `DefaultRoutingConfig`
- Injection blocking: active — refuses requests at `InjectionHigh` risk, exports counter metric
- Skill generator: wired — `skill_create` tool uses LLM to generate skill content from description
- Cron scheduler: real — `NewScheduler` instantiated in both entrypoints, NL cron via `ParseNLCron`, SQLite-persisted jobs
- MCP `AttachmentStore` + `PermissionStore`: wired — `attachment_*` and `permissions_*` MCP tools now functional
- LLM provider parity: `RegisterAll` helper, all 12 providers available in all 5 binaries
- Orphan tools registered: `image_generation`, `neutts_synth`, `session_search`
- `RegisterCustomToolTool` wired in all 5 binaries
- `/health` endpoint: unconditional `:8080/health` in `hera-agent`, answers `{"status":"ok","version":"v0.11.1"}`
- `managed_gateway.go` removed (no concrete use case in scope)

---

## v0.12.x — Sub-agent Specializations [SHIPPED]

Theme: purpose-built agent binaries for high-value workloads.

### v0.12.2 (shipped 2026-04-18)

- **Theme 1**: hera-swe PR generation via `gh` CLI — `internal/swe/prgen.go`, wired into `cmd/hera-swe/main.go`
- **Theme 2**: Platform integration test framework — `tests/integration/platforms/harness.go`, Email + Matrix live tests, 9 skeleton stubs with README docs
- **Theme 3**: Slash command gap audit — all `/` commands verified present; no gaps found
- **Theme 4**: Platform adapter smoke tests — covered by Theme 2 framework
- **Theme 5**: Release — version bump to `0.12.2`, `RELEASE_v0.12.2.md`

### v0.12.1 and earlier

See `RELEASE_v0.12.1.md` and prior release notes.

---

## v0.13.x — Multi-agent Orchestration [SHIPPED]

Theme: first-class multi-agent coordination primitives.

### v0.13.0 (shipped 2026-04-18)

- **Theme 1**: `delegate_task` tool — `internal/tools/builtin/delegate_task.go` + `AgentRegistry` in `internal/agent/delegate.go`; agents invoke other agents by name with structured prompt + optional context
- **Theme 2**: Shared agent-to-agent session bus — `internal/gateway/agent_bus.go`; pub/sub with topic-scoped channels (cap 16), non-blocking publish, `DelegationObserver` integration
- **Theme 3**: Supervisor agent + lifecycle management — `internal/supervisor/supervisor.go` + `cmd/hera-supervisor/main.go`; Spawn/Stop/Restart/StopAll with generation-capture goroutine safety, HTTP status + health endpoints
- **Theme 4**: Per-agent token and cost budget enforcement — `internal/agent/budget.go` wired into `Agent.HandleMessage`; `BudgetConfig` in `AgentConfig`; rejects with `ErrBudgetExceeded`, records usage after every LLM call
- **Theme 5**: Release — version bump to `0.13.0`, `RELEASE_v0.13.0.md`

### v0.13.1 (shipped 2026-04-18)

- **Bug fixes**: delegate_task + AgentBus wired; /health version corrected; cross-binary memory provider parity; cron in hera-acp + hera-mcp; supermemory query-based forget returns explicit error

### v0.13.2 (current)

- **Bug 1**: hera-rl system prompt corrected — removed reference to deleted `rl_list_environments` tool; documents real `rl_training` actions
- **Bug 2**: delegate_task caller identity threaded through to AgentBus From field; `WithCallerName` API added
- **Theme 2**: Stale docs corrected — ROADMAP (current) tag, README version reference, RELEASE_v0.13.1.md date
- **Theme 3**: vision.go + url_safety.go — removed fake success strings; vision returns explicit error; url_safety implements real Google Safe Browsing API v4 lookup

---

## Carried forward from earlier versions

- Integration tests for the 11 new platform adapters (WhatsApp, Signal, Matrix, etc.) — requires real credentials; deferred to v0.12.x
- `neutts_synth`: requires NeuTTS Python runtime — tool registers, executes if runtime present
- `image_generation`: requires `OPENAI_API_KEY` with DALL-E 3 access — tool registers unconditionally
- `vision`: multimodal LLM call not yet wired; tool returns explicit error until an OpenAI GPT-4o or Anthropic Claude 3+ integration is added

## v0.14.0 — Pluggable Context Engines (shipped)

15-method `plugins.ContextEngine` interface separating decide/act. `BaseContextEngine` embedded struct provides safe defaults for optional methods. Built-in `compressor` engine wraps the existing summarizer. Full agent lifecycle integration: `UpdateFromResponse` after every LLM call, `OnSessionStart`/`OnSessionEnd`/`OnSessionReset` at session boundaries, `UpdateModel` on `/model` switch. Engine-exposed tools automatically harvested into the tool registry.

Shipped engine:

- **compressor (default)** — `internal/agent.CompressorEngine`. Wraps the existing summarizer with `ProtectFirstN` (system prompt) and `ProtectLastN` (recent conversation) message preservation.

Planned future engines:

- **summarizer** — purely LLM-driven rolling summary, no FTS lookup.
- **vector-retrieval** — embed + cosine-similarity context selection from session transcript.
- **hierarchical** — multi-tier summary (turn → topic → session).
- **lcm** — lossless context management with DAG-based memory + `lcm_grep`/`lcm_describe`/`lcm_expand` tools.

Activation via `agent.compression.engine: <name>` in `hera.yaml`. Only one engine active per agent; default stays `compressor`.

---

## Versioning policy

Minor bumps (`v0.x.0`) for new capabilities. Patch bumps (`v0.x.y`) for bug fixes, wiring fixes, and documentation corrections. No breaking API changes until v1.0.
