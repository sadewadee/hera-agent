# Context Compression

Long conversations eventually fill the LLM's context window. Hera automatically compresses old messages by summarizing them, so conversations can continue indefinitely without hitting token limits.

## How it works

1. After each turn, Hera estimates the token count of the conversation history.
2. If the count exceeds `compression.threshold` × (model's context window), compression triggers.
3. Older messages (everything except the last `protected_turns` turn pairs) are summarized using the LLM.
4. The summary replaces the old messages as a single `system` message.
5. The conversation continues with the protected recent turns still intact.

## Configuration

```yaml
agent:
  compression:
    enabled: true         # enable context compression
    threshold: 0.5        # compress when context is 50% full (0.0–1.0)
    target_ratio: 0.2     # compress down to 20% of window size
    protected_turns: 5    # keep the last 5 turn pairs verbatim
    summary_model: ""     # model for summaries (empty = use default_model)
```

### What the settings mean

- **`threshold: 0.5`** — triggers at 50% of the context window. Set lower (e.g. `0.3`) for more aggressive compression; set higher (e.g. `0.8`) for less frequent compression.
- **`target_ratio: 0.2`** — after compression the kept history represents about 20% of the window. This leaves room for new messages.
- **`protected_turns: 5`** — the 5 most recent turn pairs (10 messages: 5 user + 5 assistant) are never touched. They're always sent verbatim.
- **`summary_model`** — you can use a cheaper model (e.g. `gpt-4o-mini`) for summaries to reduce cost. If empty, the default model is used.

## Token estimation

Hera estimates tokens using a simple word-count heuristic (~1.3 tokens/word). This is intentionally conservative — it may slightly over-estimate, which means compression triggers a bit early. This is safer than under-estimating and hitting a hard context limit.

## What gets summarized

Only messages older than the `protected_turns` window are summarized. The summary is prepended as:

```
[Summary of earlier conversation]
The user was asking about X. We discussed Y, I helped with Z, and the key
decisions were: ...
[End of summary]
```

The protected recent turns follow immediately after.

## Compression feedback

After compression, the agent informs the user:

```
[Context compressed: summarized 47 messages into a 280-token summary.
Keeping the last 5 turns verbatim.]
```

## Disabling compression

```yaml
agent:
  compression:
    enabled: false
```

When disabled, messages accumulate until the model's context limit is hit, at which point the LLM API will return an error.

## Using a cheap summary model

To minimize cost, use a fast/cheap model only for summaries:

```yaml
agent:
  default_model: gpt-4o
  compression:
    enabled: true
    summary_model: gpt-4o-mini  # summaries use mini, responses use 4o
```
