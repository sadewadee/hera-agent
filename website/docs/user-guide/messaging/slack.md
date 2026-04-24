---
title: Slack
sidebar_label: Slack
---

# Slack Integration

Add Hera to your Slack workspace using Socket Mode for real-time messaging without a public webhook URL.

## Prerequisites

- A Slack workspace where you have permission to install apps
- A Slack app created at [api.slack.com/apps](https://api.slack.com/apps)

## Create a Slack App

1. Go to [api.slack.com/apps](https://api.slack.com/apps) and click **Create New App → From Scratch**
2. Name your app and select your workspace
3. Under **Socket Mode**, enable it and generate an App-Level Token with `connections:write` scope — copy this token
4. Under **OAuth & Permissions**, add Bot Token Scopes:
   - `chat:write`
   - `im:history`
   - `im:read`
   - `channels:history`
   - `app_mentions:read`
5. Under **Event Subscriptions → Subscribe to bot events**, add:
   - `message.im`
   - `app_mention`
6. Install the app to your workspace and copy the Bot User OAuth Token

## Configuration

### Environment Variables

```bash
SLACK_BOT_TOKEN=xoxb-your-bot-token
SLACK_APP_TOKEN=xapp-your-app-level-token
SLACK_ALLOW_LIST=U12345678,U87654321  # Slack user IDs
```

### Config File (`~/.hera/config.yaml`)

```yaml
gateway:
  platforms:
    slack:
      enabled: true
      token: ${SLACK_BOT_TOKEN}
      extra:
        app_token: ${SLACK_APP_TOKEN}
      allow_list:
        - "U12345678"
```

## Starting Hera with Slack

```bash
hera gateway --platform slack
```

Socket Mode connects directly to Slack's WebSocket servers — no public URL required.

## Finding User IDs

In Slack, click a user's profile → **More** → **Copy member ID** to get their user ID.

## Interacting with the Bot

- Direct message the bot to start a private conversation
- Mention the bot (`@YourBotName`) in a channel to get a response
- Each DM conversation maintains its own session context

## Security

- `allow_list` restricts which Slack users can interact with the agent
- App-level tokens have limited scope (`connections:write` only)
- Socket Mode avoids the need to expose a public webhook endpoint
