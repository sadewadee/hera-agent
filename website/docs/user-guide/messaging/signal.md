---
title: Signal
sidebar_label: Signal
---

# Signal Integration

Connect Hera to Signal via the [signal-cli REST API](https://github.com/bbernhard/signal-cli-rest-api).

## Prerequisites

- A Signal phone number to use as the bot
- [signal-cli-rest-api](https://github.com/bbernhard/signal-cli-rest-api) running locally or on a server

## Set Up signal-cli REST API

The easiest way is Docker:

```bash
docker run -d \
  -p 8080:8080 \
  -v /path/to/signal-cli-config:/home/.local/share/signal-cli \
  -e MODE=normal \
  bbernhard/signal-cli-rest-api
```

Register or link a phone number following the [signal-cli documentation](https://github.com/bbernhard/signal-cli-rest-api#getting-started).

## Configuration

### Environment Variables

```bash
SIGNAL_API_URL=http://localhost:8080
SIGNAL_PHONE_NUMBER=+15551234567      # The phone number registered with signal-cli
SIGNAL_ALLOW_LIST=+15559876543        # Allowed sender phone numbers
```

### Config File (`~/.hera/config.yaml`)

```yaml
gateway:
  platforms:
    signal:
      enabled: true
      extra:
        api_url: http://localhost:8080
        phone_number: "+15551234567"
      allow_list:
        - "+15559876543"
```

## Starting Hera with Signal

```bash
hera gateway --platform signal
```

Hera polls the signal-cli REST API for new messages and responds via the same API.

## Security Considerations

- Signal messages are end-to-end encrypted by the Signal protocol
- The signal-cli REST API should not be exposed publicly — run it on localhost or a private network
- Use `allow_list` to restrict which Signal numbers can interact with the agent
- The phone number used for the bot cannot simultaneously be used on a phone for personal messaging

## Limitations

- Requires a dedicated phone number for the bot
- signal-cli is a third-party tool and not officially supported by Signal
- Group message support depends on the signal-cli version
