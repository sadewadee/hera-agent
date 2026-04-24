# Hera Plugin System

## Overview

Hera's plugin system provides extensible memory providers that integrate with
external knowledge and memory services. Each plugin implements the
`MemoryProvider` interface defined in `memory_provider.go`.

## Memory Providers

| Provider | Description | Config Env Var |
|----------|-------------|----------------|
| `holographic` | Holographic memory with local SQLite FTS5 | Built-in |
| `honcho` | Dialectic memory via Honcho API | `HONCHO_API_KEY` |
| `mem0` | Mem0 Platform with circuit breaker | `MEM0_API_KEY` |
| `openviking` | Volcengine knowledge filesystem (viking:// URIs) | `OPENVIKING_ENDPOINT` |
| `retaindb` | RetainDB cross-session cloud memory | `RETAINDB_API_KEY` |
| `supermemory` | Supermemory semantic long-term memory | `SUPERMEMORY_API_KEY` |
| `hindsight` | Hindsight context recall | `HINDSIGHT_API_KEY` |
| `byterover` | ByteRover knowledge base | `BYTEROVER_API_KEY` |

## Interface

Every memory provider must implement `plugins.MemoryProvider`:

```go
type MemoryProvider interface {
    Name() string
    IsAvailable() bool
    Initialize(sessionID string) error
    SystemPromptBlock() string
    Prefetch(query, sessionID string) string
    SyncTurn(userContent, assistantContent, sessionID string)
    OnMemoryWrite(action, target, content string)
    OnPreCompress(messages []map[string]interface{}) string
    OnSessionEnd(messages []map[string]interface{})
    GetToolSchemas() []ToolSchema
    HandleToolCall(toolName string, args map[string]interface{}) (string, error)
    GetConfigSchema() []ConfigField
    Shutdown()
}
```

## How to Activate a Memory Provider

All 8 plugin providers ship with Hera and are registered automatically at startup.
To switch from the default SQLite store to a cloud provider:

**Step 1 — Set required env vars** (provider-specific, see table above).

**Step 2 — Set `memory.provider` in `~/.hera/config.yaml`:**

```yaml
memory:
  provider: mem0   # or hindsight, holographic, honcho, byterover, openviking, retaindb, supermemory
```

**Step 3 — Restart Hera.**

The selected plugin runs as a *sidecar* alongside the built-in SQLite store.
SQLite remains the primary store (conversation history, facts, notes). The
sidecar adds cloud-augmented recall, semantic search, or cross-device sync
depending on the provider.

All providers are lazy — only the one named in config is initialized;
the rest have zero runtime cost.

## Adding a New Provider

1. Create a new directory under `plugins/memory/<name>/`.
2. Implement the `MemoryProvider` interface.
3. Add a `New()` constructor returning `*Provider`.
4. Register in the plugin registry via `init()` or explicit registration.

## Context Engines

Context engines manage conversation context: when to compress, how to compress,
and optionally what tools to expose to the agent. The built-in engine is
`compressor` (wraps `internal/agent.Compressor`). Select via
`agent.compression.engine` in `hera.yaml`.

### Adding a New Context Engine

1. Create a new directory under `plugins/context_engine/<name>/`.
2. Embed `plugins.BaseContextEngine` and implement the required methods:
   `Name()`, `IsAvailable()`, `Initialize(ContextEngineConfig)`,
   `UpdateFromResponse(llm.Usage)`, `ShouldCompress(int)`,
   `Compress(ctx, []llm.Message, int)`.
3. Override optional methods from `BaseContextEngine` as needed
   (e.g., `GetToolSchemas`/`HandleToolCall` for engine-owned tools).
4. Add a `New()` constructor and register via
   `registry.RegisterContextEngine(New())` in `internal/contextengine/bootstrap.go`.
5. Expose tools via `GetToolSchemas()`/`HandleToolCall()` — they are
   automatically harvested by `builtin.RegisterEngineTools`.

## Design Principles

- All HTTP clients use `net/http` from the standard library (no SDK dependencies).
- API keys are read from environment variables, never hardcoded.
- Goroutines are used for async operations (`SyncTurn`, `OnMemoryWrite`).
- `sync.Mutex` protects shared state.
- Structured logging via `log/slog`.
- Errors are wrapped with `fmt.Errorf` and `%w`.
