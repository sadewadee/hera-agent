# SMS Adapter Integration Tests

## Requirements

- A Twilio account with a purchased phone number, OR
- A Vonage (Nexmo) account with a virtual number
- `HERA_TEST_SMS_PROVIDER` — `twilio` or `vonage`
- `HERA_TEST_SMS_ACCOUNT_SID` — Twilio Account SID (or Vonage API key)
- `HERA_TEST_SMS_AUTH_TOKEN` — Twilio Auth Token (or Vonage API secret)
- `HERA_TEST_SMS_FROM` — Your purchased number in E.164 format
- `HERA_TEST_SMS_TO` — Recipient number in E.164 format (use a test number)

## Running

```bash
HERA_TEST_SMS_PROVIDER=twilio \
HERA_TEST_SMS_ACCOUNT_SID=ACxxxxxxxxxxxxxxxx \
HERA_TEST_SMS_AUTH_TOKEN=xxxxxxxxxxxxxxxx \
HERA_TEST_SMS_FROM=+15005550006 \
HERA_TEST_SMS_TO=+15005550006 \
go test ./tests/integration/platforms/sms/... -v
```

Note: Twilio provides test credentials and magic numbers that do not send real SMS. Use `+15005550006` as both sender and recipient for free testing.
