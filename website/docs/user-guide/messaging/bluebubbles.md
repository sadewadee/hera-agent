---
title: BlueBubbles (iMessage)
sidebar_label: BlueBubbles
---

# BlueBubbles Integration

Connect Hera to iMessage via [BlueBubbles](https://bluebubbles.app), a self-hosted iMessage server for macOS.

## Prerequisites

- A Mac running macOS with an Apple ID signed in to iMessage
- [BlueBubbles server](https://github.com/BlueBubblesApp/bluebubbles-server) installed and running on the Mac

## Set Up BlueBubbles Server

1. Download and install the [BlueBubbles server app](https://bluebubbles.app/downloads/) on your Mac
2. Open BlueBubbles and complete the setup wizard
3. Note the server's **URL** and **Password** from the settings
4. Ensure the server is accessible from where Hera runs (same machine or network)

## Configuration

### Environment Variables

```bash
BLUEBUBBLES_URL=http://localhost:1234
BLUEBUBBLES_PASSWORD=your_server_password
BLUEBUBBLES_ALLOW_LIST=+15551234567,handle@icloud.com  # iMessage handles
```

### Config File (`~/.hera/config.yaml`)

```yaml
gateway:
  platforms:
    bluebubbles:
      enabled: true
      extra:
        url: http://localhost:1234
        password: ${BLUEBUBBLES_PASSWORD}
      allow_list:
        - "+15551234567"
```

## Starting Hera with BlueBubbles

```bash
hera gateway --platform bluebubbles
```

## iMessage Handles

iMessage handles can be phone numbers (`+15551234567`) or email addresses (`user@icloud.com`). Use these in `allow_list` to control access.

## Limitations

- Requires a Mac running macOS with an active Apple ID signed in
- BlueBubbles server must be running for Hera to send/receive messages
- iMessage encryption is handled by Apple — Hera communicates with BlueBubbles server over your local network
- Group chats are supported depending on the BlueBubbles server version

## Remote Access

To access BlueBubbles from a remote server, you can:
- Use a VPN to reach the Mac
- Enable port forwarding (use HTTPS and authentication)
- Use BlueBubbles' built-in ngrok/Cloudflare Tunnel support in the server settings

## Security

- Keep the BlueBubbles server on a local network or behind a VPN when possible
- Use HTTPS if exposing the server publicly
- Set `BLUEBUBBLES_PASSWORD` — never leave it blank in production
