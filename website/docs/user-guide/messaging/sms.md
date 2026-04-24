---
title: SMS (Twilio)
sidebar_label: SMS
---

# SMS Integration

Send and receive SMS messages with Hera via [Twilio](https://www.twilio.com).

## Prerequisites

- A Twilio account
- A Twilio phone number with SMS capability

## Twilio Setup

1. Sign up at [twilio.com](https://www.twilio.com) and complete phone number verification
2. Purchase a phone number from the Twilio console with SMS capability
3. Note your **Account SID** and **Auth Token** from the console dashboard
4. Configure the phone number's messaging webhook to point to Hera

## Configuration

### Environment Variables

```bash
TWILIO_ACCOUNT_SID=ACxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
TWILIO_AUTH_TOKEN=your_auth_token
TWILIO_PHONE_NUMBER=+15551234567       # Your Twilio phone number
TWILIO_WEBHOOK_URL=https://yourdomain.com/sms  # Hera's public URL
TWILIO_ALLOW_LIST=+15559876543,+15550001111    # Allowed sender numbers
```

### Config File (`~/.hera/config.yaml`)

```yaml
gateway:
  platforms:
    sms:
      enabled: true
      extra:
        account_sid: ${TWILIO_ACCOUNT_SID}
        auth_token: ${TWILIO_AUTH_TOKEN}
        phone_number: "+15551234567"
        webhook_url: https://yourdomain.com/sms
      allow_list:
        - "+15559876543"
```

## Starting Hera with SMS

```bash
hera gateway --platform sms
```

Hera starts an HTTP server to receive Twilio webhook callbacks and respond to incoming SMS.

## Configuring the Twilio Webhook

In the Twilio console:
1. Go to **Phone Numbers → Manage → Active numbers**
2. Click your number
3. Under **Messaging**, set the **Webhook** URL to `https://yourdomain.com/sms`
4. Set the HTTP method to **POST**

## SMS Length Limits

SMS messages are limited to 160 characters per segment. Hera's responses may be split into multiple SMS segments automatically by Twilio. Long responses are recommended to be kept concise in the agent's personality configuration.

## Security

- Validate Twilio signatures to ensure requests genuinely come from Twilio
- Use `allow_list` to restrict which phone numbers can interact with the agent
- Store Auth Token in an environment variable — never commit it to version control
- Use HTTPS for the webhook URL
