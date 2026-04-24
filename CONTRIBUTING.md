# Contributing to Hera

## Getting Started

1. Fork the repository
2. Clone your fork: `git clone https://github.com/YOUR-USERNAME/hera.git`
3. Create a branch: `git checkout -b feature/my-feature`
4. Make changes and test: `go test ./...`
5. Commit and push: `git push origin feature/my-feature`
6. Open a Pull Request

## Development Setup

```bash
go mod download
go build ./...
go test ./...
```

## Coding Standards

- Follow standard Go conventions (gofmt, govet)
- Use net/http stdlib for HTTP clients (no SDK dependencies)
- Use modernc.org/sqlite for database (pure Go, no CGO)
- Write tests for all new functionality
- Keep PRs under 500 lines

## Testing

```bash
# All tests
go test ./...

# With race detection
go test -race ./...

# Specific package
go test ./internal/agent/...
```

## Pull Request Process

1. Ensure all tests pass
2. Update documentation if needed
3. Follow the PR template
4. Request review
