# Hera Project Conventions

## Language & Module
- Go 1.26.1
- Module: github.com/sadewadee/hera
- Pure Go SQLite: modernc.org/sqlite
- No CGO required

## Architecture
- All HTTP clients use net/http stdlib (no SDK deps)
- LLM providers use raw REST APIs
- Platform adapters are goroutine-per-adapter
- Tools implement tools.Tool interface
- Skills are Markdown files with YAML frontmatter

## Testing
- Use testify for assertions
- Use httptest for HTTP testing
- Use t.TempDir() for temp files
- Race detection: go test -race ./...

## Key Patterns
- Config: Viper (YAML + env vars)
- Logging: log/slog
- Errors: fmt.Errorf with %w wrapping
- Concurrency: sync.Mutex for shared state
- Streaming: channels for LLM events
