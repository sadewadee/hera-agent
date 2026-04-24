---
title: Mattermost
sidebar_label: Mattermost
---

# Mattermost Integration

Connect Hera to a Mattermost server using the Mattermost REST API.

## Prerequisites

- A Mattermost server (self-hosted or Mattermost Cloud)
- A Mattermost account with bot or user credentials
- Permission to create bots or access tokens in the workspace

## Create a Bot Account

In Mattermost (System Console → Integrations → Bot Accounts):
1. Enable bot account creation
2. Create a new bot account and copy the **Access Token**

Alternatively, create a personal access token under **Profile → Security → Personal Access Tokens**.

## Configuration

### Environment Variables

```bash
MATTERMOST_URL=https://your-mattermost-instance.com
MATTERMOST_TOKEN=your_access_token
MATTERMOST_TEAM_ID=team_id_here          # Optional: restrict to one team
MATTERMOST_CHANNEL_ID=channel_id_here   # Optional: restrict to one channel
MATTERMOST_ALLOW_LIST=user1,user2       # Mattermost user IDs
```

### Config File (`~/.hera/config.yaml`)

```yaml
gateway:
  platforms:
    mattermost:
      enabled: true
      extra:
        url: https://your-mattermost-instance.com
        token: ${MATTERMOST_TOKEN}
        team_id: team_id_here
```

## Starting Hera with Mattermost

```bash
hera gateway --platform mattermost
```

## WebSocket Connection

Hera uses Mattermost's WebSocket API for real-time event streaming. The WebSocket URL is derived automatically from the server URL (replacing `https://` with `wss://`).

## Security

- Use a bot account rather than your personal account credentials
- Store the access token in an environment variable
- Use `allow_list` to restrict which Mattermost users can interact with the agent
- Mattermost supports single sign-on (SSO) — user IDs are stable across SSO providers

## Self-Hosted Setup

For self-hosted Mattermost:
1. Ensure the REST API is accessible from Hera's host
2. Configure TLS on your Mattermost server
3. Set `MATTERMOST_URL` to the HTTPS URL of your server
