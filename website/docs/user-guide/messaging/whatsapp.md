---
title: WhatsApp
sidebar_label: WhatsApp
---

# WhatsApp Integration

Connect Hera to WhatsApp Business via the Meta Cloud API.

## Prerequisites

- A Meta Developer account
- A WhatsApp Business account linked to a Meta app
- A phone number verified for WhatsApp Business

## Setup

1. Go to [developers.facebook.com](https://developers.facebook.com)
2. Create an app and add the **WhatsApp** product
3. Under **WhatsApp → Configuration**, note your:
   - Phone number ID
   - WhatsApp Business Account ID
4. Generate a permanent token in **System Users** (or use a temporary token for testing)
5. Configure a webhook URL in the WhatsApp settings

## Configuration

### Environment Variables

```bash
WHATSAPP_PHONE_NUMBER_ID=123456789012345
WHATSAPP_ACCESS_TOKEN=EAABxxxxxxxxxxxxxx
WHATSAPP_WEBHOOK_SECRET=your-verify-token
WHATSAPP_ALLOW_LIST=+15551234567,+15559876543  # E.164 formatted numbers
```

### Config File (`~/.hera/config.yaml`)

```yaml
gateway:
  platforms:
    whatsapp:
      enabled: true
      extra:
        phone_number_id: "123456789012345"
        access_token: ${WHATSAPP_ACCESS_TOKEN}
        webhook_secret: ${WHATSAPP_WEBHOOK_SECRET}
      allow_list:
        - "+15551234567"
```

## Starting Hera with WhatsApp

Hera listens for webhook events from Meta:

```bash
hera gateway --platform whatsapp
```

Ensure your server is publicly accessible over HTTPS. Configure the webhook URL in the Meta Developer Portal to point to `https://yourdomain.com/whatsapp`.

## Webhook Verification

When you save the webhook URL in the Meta portal, Meta sends a verification request. Hera handles this automatically using the `webhook_secret` (verify token) you configured.

## Supported Message Types

- Text messages
- Images and documents (forwarded to vision/file tools)
- Voice messages (transcribed automatically)

## Security

- Store the access token in an environment variable
- Use `allow_list` to restrict which phone numbers can interact with the agent
- Rotate the webhook verify token periodically
