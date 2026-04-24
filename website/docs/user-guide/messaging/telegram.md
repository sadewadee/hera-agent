---
title: Telegram
sidebar_label: Telegram
---

# Telegram Integration

Connect Hera to Telegram to chat with your agent through Telegram messages.

## Prerequisites

- A Telegram account
- A Telegram bot created via [@BotFather](https://t.me/BotFather)

## Create a Bot

1. Open Telegram and message [@BotFather](https://t.me/BotFather)
2. Send `/newbot` and follow the prompts
3. Copy the bot token (format: `123456789:ABCdef...`)

## Configuration

### Environment Variables

```bash
TELEGRAM_BOT_TOKEN=123456789:ABCdefGHIjklMNOpqrSTUvwxYZ
TELEGRAM_ALLOW_LIST=123456789,987654321   # Telegram user IDs (comma-separated)
TELEGRAM_WEBHOOK_URL=https://yourdomain.com/telegram  # Optional: for webhook mode
TELEGRAM_WEBHOOK_SECRET=your-webhook-secret           # Optional
```

### Config File (`~/.hera/config.yaml`)

```yaml
gateway:
  platforms:
    telegram:
      enabled: true
      token: ${TELEGRAM_BOT_TOKEN}
      allow_list:
        - "123456789"
        - "987654321"
```

## Starting Hera with Telegram

```bash
hera gateway --platform telegram
```

Or run all platforms at once:

```bash
hera gateway
```

## Finding Your User ID

Send a message to [@userinfobot](https://t.me/userinfobot) on Telegram to get your numeric user ID. Add this to `allow_list` so only authorized users can interact with your agent.

## Webhook vs Polling

By default, Hera polls for new Telegram messages. To use webhooks (recommended for production):

1. Set `TELEGRAM_WEBHOOK_URL` to a publicly accessible HTTPS URL
2. Set `TELEGRAM_WEBHOOK_SECRET` to a random secure string
3. Hera will register the webhook with Telegram automatically on startup

## Security

- Always set `allow_list` to restrict who can interact with the agent
- Set `gateway.allow_all: false` in config (the default)
- Use a webhook secret to verify requests come from Telegram
- Store the bot token in an environment variable, not in the config file

## Supported Message Types

- Text messages
- File attachments (passed to vision/file tools)
- Voice messages (transcribed via the transcription tool)
- Images (processed via the vision tool)
