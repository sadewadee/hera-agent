# Installation

Hera is a self-improving, multi-platform AI agent written in Go. It has no CGO requirements and ships as a single static binary.

## Requirements

- Go 1.21+ (for `go install`)
- An API key for at least one LLM provider (OpenAI, Anthropic, Gemini, etc.)

---

## Build from Source (recommended)

```bash
git clone https://github.com/sadewadee/hera-agent
cd hera-agent
go build -o bin/hera ./cmd/hera
```

This produces the `hera` binary in `./bin/`. Available binaries:

| Binary | Purpose |
|--------|---------|
| `hera` | Interactive CLI agent |
| `hera-agent` | Headless background agent |
| `hera-mcp` | MCP server (IDE integration) |
| `hera-acp` | ACP adapter (editor integration) |
| `hera-batch` | Batch processing |
| `hera-swe` | Software engineering agent |

To build all binaries at once:

```bash
go build ./cmd/...
```

Binaries land in the current directory (or `bin/` if you specify `-o bin/`).

---

## Docker

A Docker Compose setup is provided for running Hera with all gateway adapters:

```bash
git clone https://github.com/sadewadee/hera.git
cd hera
cp env.example .env      # fill in your API keys
docker compose -f deployments/docker-compose.yml up -d
```

To run only the CLI container:

```bash
docker run --rm -it \
  -e OPENAI_API_KEY=sk-... \
  -v ~/.hera:/root/.hera \
  ghcr.io/sadewadee/hera:latest
```

---

## Nix / NixOS

A `flake.nix` is included. To build and run without installing:

```bash
nix run github:sadewadee/hera
```

To install into your profile:

```bash
nix profile install github:sadewadee/hera
```

To enter a development shell with all tooling:

```bash
nix develop
```

For NixOS system-wide installation, add to your `configuration.nix` or `flake.nix`:

```nix
{
  environment.systemPackages = [
    (builtins.getFlake "github:sadewadee/hera").packages.${system}.default
  ];
}
```

---

## Verify Installation

```bash
hera --version
# hera v0.1.0
```

---

## Next Steps

- [Quick Start](./quickstart.md) — have your first conversation in 5 minutes
- [Configuration](./configuration.md) — configure providers, memory, and more
