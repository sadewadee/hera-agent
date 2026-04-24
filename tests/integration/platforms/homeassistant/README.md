# Home Assistant Adapter Integration Tests

## Requirements

- A running Home Assistant instance (local or remote)
- A Long-Lived Access Token (Settings > Profile > Long-Lived Access Tokens)
- `HERA_TEST_HA_URL` — Base URL (e.g. `http://homeassistant.local:8123`)
- `HERA_TEST_HA_TOKEN` — Long-lived access token
- `HERA_TEST_HA_WEBHOOK_ID` — Webhook ID created in HA automations for inbound messages

## Running

```bash
HERA_TEST_HA_URL=http://homeassistant.local:8123 \
HERA_TEST_HA_TOKEN=eyJ0eXAiOiJKV1Q... \
HERA_TEST_HA_WEBHOOK_ID=hera_incoming \
go test ./tests/integration/platforms/homeassistant/... -v
```
