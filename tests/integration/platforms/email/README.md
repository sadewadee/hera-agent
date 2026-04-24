# Email Adapter Integration Tests

## What is tested

The reference integration tests in this package cover:
- Inbound webhook message receipt (`/email/incoming` endpoint) — no credentials required
- HTTP method validation and empty body rejection — no credentials required
- `Send()` error when adapter is not connected — no credentials required

The SMTP send path (sending real email) is not exercised by the automated tests because it requires a real SMTP server. To test it locally, use MailHog.

## Running the httptest-based tests (no credentials needed)

```bash
go test ./tests/integration/platforms/email/... -v
```

## Running against a real MailHog instance

1. Install and run MailHog:
   ```bash
   brew install mailhog
   mailhog &
   # SMTP on :1025, Web UI on :8025
   ```

2. Run with credentials:
   ```bash
   HERA_TEST_SMTP_HOST=localhost \
   HERA_TEST_SMTP_PORT=1025 \
   HERA_TEST_SMTP_USER= \
   HERA_TEST_SMTP_PASS= \
   HERA_TEST_FROM=test@example.com \
   go test ./tests/integration/platforms/email/... -v -run TestEmailAdapter_SMTPSend
   ```

## Required environment variables for SMTP tests

| Variable | Description | Example |
|----------|-------------|---------|
| `HERA_TEST_SMTP_HOST` | SMTP server hostname | `localhost` |
| `HERA_TEST_SMTP_PORT` | SMTP port | `1025` |
| `HERA_TEST_SMTP_USER` | SMTP username (blank for MailHog) | `""` |
| `HERA_TEST_SMTP_PASS` | SMTP password (blank for MailHog) | `""` |
| `HERA_TEST_FROM` | Sender address | `bot@example.com` |
