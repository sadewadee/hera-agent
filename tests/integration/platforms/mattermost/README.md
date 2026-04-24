# Mattermost Adapter Integration Tests

## Requirements

- A Mattermost server (self-hosted or cloud)
- A bot account with a Personal Access Token
- `HERA_TEST_MM_URL` — Mattermost server URL (e.g. `https://mattermost.example.com`)
- `HERA_TEST_MM_TOKEN` — Bot user personal access token
- `HERA_TEST_MM_TEAM` — Team name or ID to post in
- `HERA_TEST_MM_CHANNEL` — Channel name or ID to send test messages to

## Running

```bash
HERA_TEST_MM_URL=https://mattermost.example.com \
HERA_TEST_MM_TOKEN=xxx \
HERA_TEST_MM_TEAM=myteam \
HERA_TEST_MM_CHANNEL=town-square \
go test ./tests/integration/platforms/mattermost/... -v
```

## Self-hosted Mattermost with Docker

```bash
docker run -d --name mattermost-preview \
  -p 8065:8065 \
  mattermost/mattermost-preview

# Access at http://localhost:8065
# Create team, channel, and bot token via admin console
```
