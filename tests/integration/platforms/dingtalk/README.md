# DingTalk Adapter Integration Tests

## Requirements

- A DingTalk developer account at open.dingtalk.com
- An app with robot capability enabled
- `HERA_TEST_DINGTALK_APP_KEY` — App Key from app credentials
- `HERA_TEST_DINGTALK_APP_SECRET` — App Secret from app credentials
- `HERA_TEST_DINGTALK_ROBOT_CODE` — Robot Code from robot configuration
- `HERA_TEST_DINGTALK_TOKEN` — Token configured for webhook signature validation
- `HERA_TEST_DINGTALK_CHAT_ID` — A chat/conversation ID to send test messages to

## Running

```bash
HERA_TEST_DINGTALK_APP_KEY=ding... \
HERA_TEST_DINGTALK_APP_SECRET=xxx \
HERA_TEST_DINGTALK_ROBOT_CODE=xxx \
HERA_TEST_DINGTALK_TOKEN=xxx \
HERA_TEST_DINGTALK_CHAT_ID=xxx \
go test ./tests/integration/platforms/dingtalk/... -v
```
