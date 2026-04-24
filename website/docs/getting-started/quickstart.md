# Quick Start

Get Hera running and have your first conversation in under 5 minutes.

## Step 1: Install

```bash
git clone https://github.com/sadewadee/hera-agent
cd hera-agent
go build -o bin/hera ./cmd/hera
export PATH="$(pwd)/bin:$PATH"   # or copy ./bin/hera to a directory already in your PATH
```

## Step 2: Set your API key

Hera reads API keys from environment variables. The simplest setup uses OpenAI:

```bash
export OPENAI_API_KEY=sk-...
```

Or add it to your shell profile (`~/.bashrc`, `~/.zshrc`, etc.) so it persists:

```bash
echo 'export OPENAI_API_KEY=sk-...' >> ~/.zshrc
source ~/.zshrc
```

Hera supports 15+ providers. See the [configuration guide](./configuration.md) for Anthropic, Gemini, Mistral, Ollama, and more.

## Step 3: Run setup

```bash
hera setup
```

This creates `~/.hera/` with default configuration files including `config.yaml` and `SOUL.md`.

## Step 4: Start chatting

```bash
hera chat
```

You'll see a prompt. Type a message and press Enter:

```
You: What can you do?

Hera: I'm an AI agent with access to tools for file operations, web search,
code execution, memory, and more. I can help you with coding tasks, research,
system administration, and general questions. What would you like to work on?

You: Search the web for the current Go version

Hera: [using web_search]
The current stable release of Go is 1.23.4, released in December 2024...
```

Press `Ctrl+C` or type `exit` to quit.

## Step 5: Explore features

### Use memory

Hera remembers things across conversations using SQLite:

```
You: Remember that my project uses PostgreSQL 16 and is deployed on AWS

Hera: Got it. I'll remember that your project uses PostgreSQL 16 on AWS.

You: (next session) What database does my project use?

Hera: Based on what you told me earlier, your project uses PostgreSQL 16,
deployed on AWS.
```

### Use a skill

Skills are Markdown documents with instructions. Hera has 170+ bundled skills:

```
You: /godmode

Hera: Godmode activated. I'll work autonomously with minimal interruptions...
```

### Try a tool

```
You: Read the contents of ./README.md and summarize it

Hera: [using read_file]
The README describes Hera, a self-improving multi-platform AI agent...
```

## Common flags

```bash
# Use a specific provider/model for this session
hera chat --provider anthropic --model claude-sonnet-4-20250514

# Start with a specific personality
hera chat --personality coder

# Non-interactive: pipe input
echo "What is 2+2?" | hera chat --no-tty
```

## Next Steps

- [Configuration](./configuration.md) — set up providers, memory, gateway
- [Features Overview](../user-guide/features/overview.md) — everything Hera can do
- [Tool Reference](../reference/tool-reference.md) — all built-in tools
