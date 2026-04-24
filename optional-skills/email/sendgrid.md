---
name: sendgrid
description: "SendGrid email API integration"
version: "1.0"
trigger: "sendgrid email api transactional"
platforms: []
requires_tools: ["run_command"]
---

# SendGrid email API integration

## Purpose
SendGrid email API integration for transactional and marketing email delivery at scale.

## Instructions
1. Set up API credentials and domain verification
2. Configure sending domain with SPF, DKIM, and DMARC
3. Implement email sending with templates
4. Set up webhooks for delivery events
5. Monitor deliverability and engagement metrics

## Domain Setup
- Verify sending domain ownership
- Configure DNS records (SPF, DKIM, DMARC)
- Set up custom return-path for bounce handling
- Warm up new sending domains gradually
- Configure IP pools for reputation isolation

## Sending
- Use templates for consistent formatting
- Implement proper error handling and retries
- Set appropriate headers (List-Unsubscribe, etc.)
- Handle bounces and complaints programmatically
- Track delivery, open, and click events

## Webhooks
- Process delivery notifications
- Handle bounce and complaint events
- Track opens and clicks
- Update recipient status based on events
- Store events for analytics and debugging

## Best Practices
- Validate email addresses before sending
- Implement exponential backoff for API rate limits
- Monitor bounce and complaint rates daily
- Maintain a clean recipient list
- Test emails with mail-tester.com before campaigns
