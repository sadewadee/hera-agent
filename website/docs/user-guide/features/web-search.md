# Web Search

Hera can search the web using the Exa API. The `web_search` tool returns current information on any topic, and `web_extract` retrieves the full text content of a specific URL.

## Setup

Get an API key from [exa.ai](https://exa.ai) and set it:

```bash
export EXA_API_KEY=your-key-here
```

Or in `~/.hera/config.yaml`:

```yaml
# Add to your environment file or pass directly
# Hera reads EXA_API_KEY from the environment automatically
```

## Usage

The agent calls `web_search` automatically when it needs current information:

```
You: What's the current price of Bitcoin?

Hera: [using web_search: "Bitcoin price today"]
As of April 2026, Bitcoin is trading at approximately $...
```

You can also ask explicitly:

```
You: Search the web for the latest Go 1.26 release notes

Hera: [using web_search: "Go 1.26 release notes"]
...
```

## web_search tool

**Tool name:** `web_search`

**Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `query` | string | yes | Search query |
| `max_results` | integer | no | Max results to return (default: 5) |

**Example tool call:**

```json
{
  "name": "web_search",
  "arguments": {
    "query": "Go 1.26 release notes",
    "max_results": 5
  }
}
```

## web_extract tool

Retrieves and parses the full text of a specific URL:

**Tool name:** `web_extract`

**Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `url` | string | yes | URL to fetch and extract text from |

**Example:**

```
You: Extract the content from https://go.dev/blog/go1.26

Hera: [using web_extract: "https://go.dev/blog/go1.26"]
Go 1.26 was released on...
```

## When web search is used

The agent decides autonomously when web search is needed. Common triggers:

- "What is the current..."
- "Latest news on..."
- "What happened today..."
- "Current price of..."
- "Recent changes to..."
- Any question about events after the model's training cutoff

## URL safety

Before fetching any URL, Hera checks it against known malicious domain lists. Blocked URLs are refused with an explanation.

## Without an Exa API key

If no `EXA_API_KEY` is set, the `web_search` tool is registered but returns an error when called. The agent will tell you that web search is unavailable. Set the key to enable it.
