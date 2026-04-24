# Feishu / Lark Adapter Integration Tests

## Requirements

- A Feishu or Lark developer account at open.feishu.cn / open.larksuite.com
- A bot app with messaging:write permission
- `HERA_TEST_FEISHU_APP_ID` — App ID from app credentials
- `HERA_TEST_FEISHU_APP_SECRET` — App Secret from app credentials
- `HERA_TEST_FEISHU_VERIFICATION_TOKEN` — Verification token for event subscription
- `HERA_TEST_FEISHU_ENCRYPT_KEY` — Optional encrypt key for event encryption
- `HERA_TEST_FEISHU_CHAT_ID` — An open_chat_id or user open_id to send test messages to

## Running

```bash
HERA_TEST_FEISHU_APP_ID=cli_xxx \
HERA_TEST_FEISHU_APP_SECRET=xxx \
HERA_TEST_FEISHU_VERIFICATION_TOKEN=xxx \
HERA_TEST_FEISHU_CHAT_ID=oc_xxx \
go test ./tests/integration/platforms/feishu/... -v
```
