---
title: Home Assistant
sidebar_label: Home Assistant
---

# Home Assistant Integration

Connect Hera to [Home Assistant](https://www.home-assistant.io) to chat with your agent through the Home Assistant companion app or notification system.

## Prerequisites

- A running Home Assistant instance (version 2023.x or later)
- A Long-Lived Access Token from your Home Assistant profile

## Generate an Access Token

1. Open Home Assistant in your browser
2. Click your profile icon in the bottom left
3. Scroll down to **Long-Lived Access Tokens**
4. Click **Create Token**, give it a name (e.g., "Hera"), and copy the token

## Configuration

### Environment Variables

```bash
HOMEASSISTANT_URL=http://homeassistant.local:8123
HOMEASSISTANT_TOKEN=eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
HOMEASSISTANT_WEBHOOK_ID=hera_webhook_abc123  # Optional: for receiving messages
HOMEASSISTANT_ALLOW_LIST=user1@example.com    # HA usernames
```

### Config File (`~/.hera/config.yaml`)

```yaml
gateway:
  platforms:
    homeassistant:
      enabled: true
      extra:
        url: http://homeassistant.local:8123
        token: ${HOMEASSISTANT_TOKEN}
        webhook_id: hera_webhook_abc123
```

## Starting Hera with Home Assistant

```bash
hera gateway --platform homeassistant
```

## Webhook Setup

To receive messages from Home Assistant (e.g., from automations or the companion app):

1. In Home Assistant, create an automation that sends a webhook to Hera
2. Use the webhook ID you configured as `webhook_id`
3. Configure the automation to POST message content to `http://hera-host:port/homeassistant`

## Use Cases

- Ask about smart home state ("Is the front door locked?")
- Trigger automations via natural language ("Turn off all lights in 30 minutes")
- Receive alerts and respond to them conversationally
- Get a daily briefing from your smart home

## Security

- Use HTTPS for your Home Assistant URL if accessing over the internet
- Long-lived access tokens have full account access — store them securely
- Consider creating a dedicated Home Assistant user with limited permissions for the Hera token
