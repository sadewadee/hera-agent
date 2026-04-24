# Matrix Adapter Integration Tests

## What is tested

The reference integration tests in this package cover:
- `Connect()` success with mock homeserver returning 200 to `/whoami` — no credentials required
- `Connect()` failure with 401 Unauthorized — no credentials required
- `Connect()` failure with missing HomeserverURL — no credentials required
- `Send()` posting to the Matrix send API — no credentials required
- `Send()` error when not connected — no credentials required

All tests use `httptest.Server` to mock the Matrix Client-Server API. No real Matrix homeserver is required.

## Running the httptest-based tests (no credentials needed)

```bash
go test ./tests/integration/platforms/matrix/... -v
```

## Running against a real Matrix homeserver (e.g. Synapse)

1. Set up a Synapse test instance:
   ```bash
   docker run -d --name synapse \
     -e SYNAPSE_SERVER_NAME=localhost \
     -e SYNAPSE_REPORT_STATS=no \
     -p 8448:8448 \
     matrixdotorg/synapse:latest generate
   ```

2. Create a bot user and get an access token via the admin API.

3. Run with credentials:
   ```bash
   HERA_TEST_MATRIX_URL=http://localhost:8448 \
   HERA_TEST_MATRIX_USER=@hera-bot:localhost \
   HERA_TEST_MATRIX_TOKEN=<access_token> \
   HERA_TEST_MATRIX_ROOM=!<room_id>:localhost \
   go test ./tests/integration/platforms/matrix/... -v -run TestMatrixAdapter_Real
   ```

## Required environment variables for real Matrix tests

| Variable | Description | Example |
|----------|-------------|---------|
| `HERA_TEST_MATRIX_URL` | Homeserver URL | `https://matrix.example.com` |
| `HERA_TEST_MATRIX_USER` | Bot user ID | `@hera:example.com` |
| `HERA_TEST_MATRIX_TOKEN` | Access token | `syt_abc123...` |
| `HERA_TEST_MATRIX_ROOM` | Test room ID | `!abc123:example.com` |
