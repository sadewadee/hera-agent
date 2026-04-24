---
title: Feishu / Lark
sidebar_label: Feishu / Lark
---

# Feishu / Lark Integration

Connect Hera to Feishu (飞书) or its international equivalent, Lark.

## Prerequisites

- A Feishu/Lark account with developer permissions
- A Feishu app created at [open.feishu.cn](https://open.feishu.cn) (Feishu) or [open.larksuite.com](https://open.larksuite.com) (Lark)

## Create a Feishu App

1. Log in to the [Feishu Open Platform](https://open.feishu.cn)
2. Go to **Developer Console → Create Application → Custom Application**
3. Note the **App ID** and **App Secret**
4. Under **Event Subscriptions**, configure a webhook URL and note the **Verification Token** and **Encrypt Key**
5. Subscribe to `im.message.receive_v1` events
6. Under **Bot**, enable the bot capability
7. Add required permissions: `im:message`, `im:message:send_as_bot`
8. Publish the application to your organization

## Configuration

### Environment Variables

```bash
FEISHU_APP_ID=cli_xxxxxxxxxxxxxxxx
FEISHU_APP_SECRET=your_app_secret
FEISHU_VERIFICATION_TOKEN=your_verification_token
FEISHU_ENCRYPT_KEY=your_encrypt_key   # Optional: enables message encryption
FEISHU_ALLOW_LIST=ou_user1,ou_user2   # Feishu Open User IDs
```

### Config File (`~/.hera/config.yaml`)

```yaml
gateway:
  platforms:
    feishu:
      enabled: true
      extra:
        app_id: ${FEISHU_APP_ID}
        app_secret: ${FEISHU_APP_SECRET}
        verification_token: ${FEISHU_VERIFICATION_TOKEN}
        encrypt_key: ${FEISHU_ENCRYPT_KEY}
```

## Starting Hera with Feishu

```bash
hera gateway --platform feishu
```

Hera must be accessible via a public HTTPS URL for Feishu to deliver events.

## Message Encryption

Feishu supports optional message encryption via AES. Set `FEISHU_ENCRYPT_KEY` to enable this. Hera handles decryption automatically.

## Rich Text Support

Feishu supports rich text (post) messages. Hera sends markdown-formatted responses as interactive messages where supported.

## Security

- Store App Secret and keys in environment variables
- Enable message encryption in production (`encrypt_key`)
- Use `allow_list` to restrict which Feishu users can interact with the agent
- Feishu verifies event signatures using the verification token — Hera validates these automatically
