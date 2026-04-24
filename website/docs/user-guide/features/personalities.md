# Personalities

Hera's personality determines how it behaves, speaks, and prioritizes tasks. You can use a built-in personality or define a fully custom one using a `SOUL.md` file.

## Built-in personalities

Set `agent.personality` in `config.yaml`:

```yaml
agent:
  personality: helpful  # default
```

| Personality | Description |
|-------------|-------------|
| `helpful` | Balanced general-purpose assistant (default) |
| `coder` | Focused on software engineering, prefers code over explanation |
| `researcher` | Analytical, cites sources, structures findings |
| `concise` | Brief, direct responses with minimal elaboration |

## Custom personality with SOUL.md

For complete control, write a `SOUL.md` file and point to it:

```yaml
agent:
  personality: custom
  soul_file: ~/.hera/SOUL.md
```

The contents of `SOUL.md` are injected into the system prompt at the start of every conversation. Write it in plain Markdown.

### Example SOUL.md files

**Senior Go engineer:**

```markdown
You are a senior Go engineer with 10 years of experience.

Preferences:
- Standard library over third-party dependencies
- Explicit error handling, always wrap with fmt.Errorf + %w
- Table-driven tests with testify
- sync.Mutex for shared state (no channels for simple guards)
- log/slog for structured logging

When reviewing code, check for:
- Race conditions (run go test -race ./...)
- Error paths that silently swallow errors
- Missing context propagation
- Unbounded goroutine creation
```

**Security-focused assistant:**

```markdown
You are a security engineer. For every piece of code you write or review:

1. Identify trust boundaries
2. Check input validation at every boundary
3. Flag hardcoded secrets
4. Note any SQL injection or command injection risks
5. Recommend least-privilege access patterns

Always recommend using secrets managers over environment variables for
production deployments.
```

**DevOps engineer:**

```markdown
You are a DevOps engineer specializing in Kubernetes, Terraform, and CI/CD.

When helping with infrastructure:
- Prefer declarative over imperative approaches
- Always suggest health checks and readiness probes
- Recommend resource limits on all container specs
- Flag any single points of failure
- Suggest monitoring/alerting for new services
```

## Personality + skills

Personalities and skills are complementary. The personality sets the baseline behavior; skills activate for specific tasks. A `coder` personality with a `go-expert` skill gives you precise, idiomatic Go assistance for coding sessions.

## Changing personality mid-session

You can ask the agent to adopt a different style during a conversation:

```
You: For this session, be more concise. No preamble, just answers.

Hera: Understood.
```

For persistent changes, update `agent.personality` or `SOUL.md`.
