---
title: DingTalk
sidebar_label: DingTalk
---

# DingTalk Integration

Connect Hera to DingTalk (钉钉), the enterprise collaboration platform popular in China.

## Prerequisites

- A DingTalk developer account at [open.dingtalk.com](https://open.dingtalk.com)
- A DingTalk organization (corp)

## Create a DingTalk App

1. Log in to the [DingTalk Open Platform](https://open.dingtalk.com)
2. Go to **Application Development → Enterprise Internal Applications → H5 Micro Application** or **Robot**
3. Create a new application and note the **AppKey** and **AppSecret**
4. Under **Message and notification → Robot**, configure the robot and get the **Robot Code**
5. For group webhook bots, create a custom robot in a group and copy the **Webhook token**

## Configuration

### Environment Variables

```bash
DINGTALK_APP_KEY=dingxxxxxxxxxxxxxxxx
DINGTALK_APP_SECRET=your_app_secret
DINGTALK_ROBOT_CODE=your_robot_code
DINGTALK_WEBHOOK_TOKEN=your_webhook_token  # For group webhook bots
DINGTALK_ALLOW_LIST=user1,user2            # DingTalk user IDs
```

### Config File (`~/.hera/config.yaml`)

```yaml
gateway:
  platforms:
    dingtalk:
      enabled: true
      extra:
        app_key: ${DINGTALK_APP_KEY}
        app_secret: ${DINGTALK_APP_SECRET}
        robot_code: ${DINGTALK_ROBOT_CODE}
```

## Starting Hera with DingTalk

```bash
hera gateway --platform dingtalk
```

## Message Types

DingTalk supports rich message types:
- **Text**: plain text messages
- **Markdown**: formatted text with links and code blocks
- **ActionCard**: interactive cards with buttons

Hera sends responses as markdown messages when the content contains formatting.

## Security

- Store AppSecret and tokens in environment variables
- The DingTalk platform signs webhook requests — Hera verifies signatures automatically
- Use `allow_list` to restrict which DingTalk user IDs can interact with the agent
