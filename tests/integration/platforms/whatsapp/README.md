# WhatsApp Adapter Integration Tests

## Requirements

- Meta WhatsApp Business API access
- A verified WhatsApp Business phone number
- An app with `whatsapp_business_messaging` permission
- `HERA_TEST_WHATSAPP_PHONE_ID` — Phone Number ID from Meta Business Manager
- `HERA_TEST_WHATSAPP_TOKEN` — Permanent system user access token
- `HERA_TEST_WHATSAPP_VERIFY_TOKEN` — Webhook verify token you set in App Dashboard
- `HERA_TEST_WHATSAPP_TO` — Recipient phone number in E.164 format

## Running

```bash
HERA_TEST_WHATSAPP_PHONE_ID=<id> \
HERA_TEST_WHATSAPP_TOKEN=<token> \
HERA_TEST_WHATSAPP_VERIFY_TOKEN=<verify> \
HERA_TEST_WHATSAPP_TO=+1234567890 \
go test ./tests/integration/platforms/whatsapp/... -v
```
