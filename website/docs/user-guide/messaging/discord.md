---
title: Discord
sidebar_label: Discord
---

# Discord Integration

Run Hera as a Discord bot in your server.

## Prerequisites

- A Discord account with permission to add bots to a server
- A Discord application and bot created at [discord.com/developers](https://discord.com/developers/applications)

## Create a Discord Bot

1. Go to the [Discord Developer Portal](https://discord.com/developers/applications)
2. Click **New Application** and give it a name
3. Navigate to **Bot** → **Add Bot**
4. Under **Privileged Gateway Intents**, enable:
   - **Message Content Intent** (required to read messages)
5. Copy the bot token
6. Navigate to **OAuth2 → URL Generator**, select `bot` scope and `Send Messages`, `Read Message History` permissions
7. Use the generated URL to invite the bot to your server

## Configuration

### Environment Variables

```bash
DISCORD_BOT_TOKEN=your-discord-bot-token
DISCORD_GUILD_ID=123456789012345678     # Optional: restrict to one server
DISCORD_ALLOW_LIST=111111111,222222222  # Discord user IDs (comma-separated)
```

### Config File (`~/.hera/config.yaml`)

```yaml
gateway:
  platforms:
    discord:
      enabled: true
      token: ${DISCORD_BOT_TOKEN}
      allow_list:
        - "111111111111111111"
```

## Starting Hera with Discord

```bash
hera gateway --platform discord
```

## Finding User and Server IDs

Enable **Developer Mode** in Discord settings (Settings → Advanced → Developer Mode). Then right-click any user or server to copy their ID.

## Interacting with the Bot

- Mention the bot or send a direct message to start a conversation
- The bot responds to all messages in channels it has access to, filtered by `allow_list`
- Each Discord channel maintains an independent session context

## Security

- Set `allow_list` to restrict which Discord user IDs can interact with the agent
- Use `DISCORD_GUILD_ID` to lock the bot to a specific server
- Never commit the bot token to version control
