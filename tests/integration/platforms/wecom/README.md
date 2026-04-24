# WeCom (WeChat Work) Adapter Integration Tests

## Requirements

- A WeCom (WeChat Work) enterprise account
- An agent application within the enterprise
- `HERA_TEST_WECOM_CORP_ID` — Enterprise Corp ID
- `HERA_TEST_WECOM_AGENT_ID` — Agent ID (numeric)
- `HERA_TEST_WECOM_SECRET` — Agent Secret
- `HERA_TEST_WECOM_TOKEN` — Token for callback message validation
- `HERA_TEST_WECOM_ENCODING_AES_KEY` — EncodingAESKey for callback encryption
- `HERA_TEST_WECOM_TO_USER` — User ID to send test messages to (`@all` or specific user)

## Running

```bash
HERA_TEST_WECOM_CORP_ID=ww... \
HERA_TEST_WECOM_AGENT_ID=1000002 \
HERA_TEST_WECOM_SECRET=xxx \
HERA_TEST_WECOM_TOKEN=xxx \
HERA_TEST_WECOM_ENCODING_AES_KEY=xxx \
HERA_TEST_WECOM_TO_USER=user123 \
go test ./tests/integration/platforms/wecom/... -v
```
