# BlueBubbles Adapter Integration Tests

## Requirements

- A Mac running macOS with iMessage signed in
- [BlueBubbles Server](https://bluebubbles.app) installed and running
- `HERA_TEST_BB_SERVER_URL` — BlueBubbles server URL (e.g. `http://localhost:1234`)
- `HERA_TEST_BB_PASSWORD` — BlueBubbles server password
- `HERA_TEST_BB_HANDLE` — iMessage handle to send test messages to (e.g. `+1234567890`)

## Running

```bash
HERA_TEST_BB_SERVER_URL=http://localhost:1234 \
HERA_TEST_BB_PASSWORD=mypassword \
HERA_TEST_BB_HANDLE=+1234567890 \
go test ./tests/integration/platforms/bluebubbles/... -v
```

## Note

BlueBubbles is macOS-only. These tests can only be run on a Mac with the
BlueBubbles server running. CI environments will not have this available.
