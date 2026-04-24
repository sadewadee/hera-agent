# Signal Adapter Integration Tests

## Requirements

- A running [signal-cli REST API](https://github.com/bbernhard/signal-cli-rest-api) instance
- A registered Signal phone number linked to the CLI
- `HERA_TEST_SIGNAL_API_URL` — Base URL of the signal-cli REST API (e.g. `http://localhost:8080`)
- `HERA_TEST_SIGNAL_NUMBER` — Sender phone number in E.164 format (e.g. `+1234567890`)
- `HERA_TEST_SIGNAL_RECIPIENT` — Recipient phone number in E.164 format

## Setup

```bash
docker run -d --name signal-cli-rest \
  -p 8080:8080 \
  -v $(pwd)/signal-cli-config:/home/.local/share/signal-cli \
  -e 'MODE=json-rpc' \
  bbernhard/signal-cli-rest-api:latest
```

Register your number via the REST API, then run:

```bash
HERA_TEST_SIGNAL_API_URL=http://localhost:8080 \
HERA_TEST_SIGNAL_NUMBER=+1234567890 \
HERA_TEST_SIGNAL_RECIPIENT=+0987654321 \
go test ./tests/integration/platforms/signal/... -v
```
