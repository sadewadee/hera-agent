---
title: Matrix
sidebar_label: Matrix
---

# Matrix Integration

Connect Hera to the Matrix protocol, compatible with Element, Cinny, and other Matrix clients.

## Prerequisites

- A Matrix homeserver account (matrix.org, your own homeserver, etc.)
- A dedicated Matrix account for the bot

## Create a Bot Account

Register a new Matrix account for the bot at your homeserver. For matrix.org, visit [app.element.io](https://app.element.io) and register.

Get an access token:

```bash
curl -XPOST "https://matrix.org/_matrix/client/v3/login" \
  -H "Content-Type: application/json" \
  -d '{"type":"m.login.password","user":"@yourbot:matrix.org","password":"yourpassword"}'
```

Copy the `access_token` from the response.

## Configuration

### Environment Variables

```bash
MATRIX_HOMESERVER_URL=https://matrix.org
MATRIX_ACCESS_TOKEN=syt_your_access_token
MATRIX_DEVICE_ID=HERA_DEVICE
MATRIX_ALLOW_LIST=@alice:matrix.org,@bob:matrix.org
```

### Config File (`~/.hera/config.yaml`)

```yaml
gateway:
  platforms:
    matrix:
      enabled: true
      extra:
        homeserver_url: https://matrix.org
        access_token: ${MATRIX_ACCESS_TOKEN}
        device_id: HERA_DEVICE
      allow_list:
        - "@alice:matrix.org"
```

## Starting Hera with Matrix

```bash
hera gateway --platform matrix
```

Hera connects using the Matrix client SDK and syncs messages in real time.

## Inviting the Bot to a Room

Once running, invite the bot account to a Matrix room:
- In Element: **Invite** → type the bot's Matrix ID (e.g., `@herabot:matrix.org`)
- The bot accepts invitations automatically and joins the room

## Security

- Store the access token securely — it grants full account access
- Use `allow_list` to restrict which Matrix users can interact with the agent
- Consider running the bot on your own homeserver for additional privacy
- Revoke the access token in account settings if the bot is compromised

## Self-Hosted Homeservers

If you run your own homeserver (Synapse, Dendrite, Conduit), set `MATRIX_HOMESERVER_URL` to your homeserver URL. The bot works the same way regardless of the homeserver.
