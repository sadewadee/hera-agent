---
name: postmark
description: "Postmark transactional email delivery"
version: "1.0"
trigger: "postmark email transactional delivery"
platforms: []
requires_tools: ["run_command"]
---

# Postmark Email

## Purpose
Send transactional emails with Postmark for reliable delivery and detailed tracking.

## Instructions
1. Set up Postmark server and API token
2. Configure email templates
3. Implement sending with the API
4. Set up webhooks for delivery events
5. Monitor deliverability and bounce handling

## API Usage
```bash
curl "https://api.postmarkapp.com/email" \
  -H "Accept: application/json" \
  -H "X-Postmark-Server-Token: YOUR_TOKEN" \
  -d '{
    "From": "sender@example.com",
    "To": "recipient@example.com",
    "Subject": "Hello",
    "HtmlBody": "<html><body>Hello!</body></html>"
  }'
```

## Best Practices
- Use templates for consistent formatting
- Handle bounces and complaints promptly
- Monitor delivery rate daily
- Use message streams for different email types
- Keep sender reputation clean
