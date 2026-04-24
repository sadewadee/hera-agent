# Hera Agent Architecture

## Agent Loop

The core agent loop in `internal/agent/agent.go`:

1. Receive user message
2. Detect prompt injection (log warning)
3. Classify query complexity (smart routing)
4. Build system prompt (identity + hints + memory + tools + skills)
5. Apply prompt caching (Anthropic)
6. Send to LLM with tool definitions
7. Execute tool calls in loop (max N iterations)
8. Generate title (first turn)
9. Redact PII before memory save
10. Record RL trajectory (optional)
11. Return response

## Tool System

Tools implement `tools.Tool` interface:
- `Name()` - tool identifier
- `Description()` - for LLM function calling
- `Parameters()` - JSON Schema
- `Execute(ctx, args)` - execution

## Memory System

SQLite with FTS5 full-text search:
- Conversation history
- User facts
- Cross-session recall
- LLM-powered summarization

## Session Management

Sessions are per-platform-per-user:
- Branching and forking
- Turn counting
- Title generation
- Context compression
