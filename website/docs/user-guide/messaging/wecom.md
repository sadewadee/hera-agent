---
title: WeCom (WeChat Work)
sidebar_label: WeCom
---

# WeCom Integration

Connect Hera to WeCom (企业微信), the enterprise version of WeChat.

## Prerequisites

- A WeCom organization (corp) account
- Admin access to your WeCom organization

## Create a WeCom App

1. Log in to the [WeCom Admin Console](https://work.weixin.qq.com/wework_admin/frame)
2. Go to **Applications → Custom Applications → Create Application**
3. Note the **AgentId** and **Secret**
4. Note the **CorpId** from the company info page
5. Under **API reception**, configure the callback URL and set the **Token** and **EncodingAESKey**

## Configuration

### Environment Variables

```bash
WECOM_CORP_ID=ww_your_corp_id
WECOM_CORP_SECRET=your_corp_secret
WECOM_AGENT_ID=1000002
WECOM_CALLBACK_TOKEN=your_callback_token
WECOM_CALLBACK_AES_KEY=your_43_char_aes_key
WECOM_ALLOW_LIST=userid1,userid2       # WeCom user IDs
```

### Config File (`~/.hera/config.yaml`)

```yaml
gateway:
  platforms:
    wecom:
      enabled: true
      extra:
        corp_id: ${WECOM_CORP_ID}
        corp_secret: ${WECOM_CORP_SECRET}
        agent_id: "1000002"
        callback_token: ${WECOM_CALLBACK_TOKEN}
        callback_aes_key: ${WECOM_CALLBACK_AES_KEY}
```

## Starting Hera with WeCom

```bash
hera gateway --platform wecom
```

Hera handles the WeCom message encryption/decryption (AES-256-CBC) automatically.

## Message Verification

WeCom sends an HTTP GET request with an `echostr` parameter to verify the callback URL. Hera responds correctly to this verification request automatically.

## Message Types

- Text messages
- Image messages (forwarded to vision tools)
- File attachments

## Security

- WeCom encrypts all messages using AES — Hera decrypts them automatically using your `callback_aes_key`
- Messages are signature-verified using your `callback_token`
- Store all secrets in environment variables
- Use `allow_list` to restrict which WeCom users can interact with the agent
