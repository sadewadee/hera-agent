---
name: webhook_feeds
description: "Webhook-based notification feeds"
version: "1.0"
trigger: "webhook notification feed event"
platforms: []
requires_tools: ["run_command"]
---

# Webhook Feeds

## Purpose
Set up and manage webhook-based notification feeds for real-time event processing.

## Instructions
1. Define webhook endpoints for event sources
2. Configure event filtering and routing
3. Implement payload processing and validation
4. Set up retry logic for failed deliveries
5. Monitor delivery rates and latency

## Webhook Security
- Validate webhook signatures (HMAC)
- Use HTTPS for all endpoints
- Implement rate limiting on receivers
- Timeout long-running webhook handlers
- Log all received webhooks for debugging

## Best Practices
- Respond quickly (< 5 seconds) to webhook requests
- Process asynchronously for complex operations
- Implement idempotency for duplicate deliveries
- Monitor for missed events and gaps
- Set up dead letter queues for failed processing
