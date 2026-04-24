# Memory

Hera has a persistent memory system backed by SQLite with FTS5 (full-text search). Memories survive across sessions and are automatically surfaced when relevant.

## How it works

Hera stores facts as key-value entries in a local SQLite database (`~/.hera/hera.db`). The agent uses the `memory` tool to:

- **Store** important facts from conversations
- **Retrieve** specific memories by key
- **Search** memories by keyword using FTS5 full-text search

## Memory nudges

Every `agent.memory_nudge_interval` turns (default: 10), Hera automatically queries the memory store for entries relevant to the current conversation and injects them into context. You don't need to ask — it happens automatically.

```yaml
agent:
  memory_nudge_interval: 10  # surface memories every 10 turns
```

## Configuration

```yaml
memory:
  provider: sqlite            # only SQLite is supported
  db_path: ~/.hera/hera.db   # database file location
  max_results: 10             # max memories returned per search query
```

The SQLite provider uses WAL mode for safe concurrent access and a 5-second busy timeout.

## Database schema

Hera creates two tables automatically:

```sql
CREATE TABLE memories (
    id        TEXT PRIMARY KEY,
    key       TEXT NOT NULL UNIQUE,
    value     TEXT NOT NULL,
    tags      TEXT,           -- JSON array of strings
    source    TEXT,           -- which platform/session created this
    created_at DATETIME,
    updated_at DATETIME
);

-- FTS5 virtual table for full-text search
CREATE VIRTUAL TABLE memories_fts USING fts5(
    key, value, tags,
    content='memories'
);
```

## Using memory from the CLI

The agent can manage its own memory via tool calls. You can also prompt it directly:

```
You: Remember that the production database password is in 1Password under "prod-db"

Hera: [using memory tool: store]
Stored: "production database password is in 1Password under prod-db"

You: Where is the production database password?

Hera: [using memory tool: search]
Based on what you've told me: the production database password is stored in
1Password under "prod-db".
```

## Memory sessions

Each conversation session also stores messages in SQLite, enabling:
- Session search (`session_search` tool) — find past conversations
- Context continuity across restarts

## Privacy

Memories are stored locally on your machine in `~/.hera/hera.db`. Nothing is sent to external services. If you enable `security.redact_pii`, PII is stripped from messages before storage.
