---
name: telegram_bot
description: "Telegram bot development"
version: "1.0"
trigger: "telegram bot api messaging"
platforms: []
requires_tools: ["run_command"]
---

# Telegram bot development

## Purpose
Telegram bot development for automated messaging and communication platform integration.

## Instructions
1. Set up API credentials and bot registration
2. Define command handlers and event listeners
3. Implement message processing and response logic
4. Configure permissions and rate limiting
5. Deploy and monitor bot health

## Bot Architecture
- Event-driven message handling
- Command parsing and routing
- Rate limiting and queue management
- Persistent storage for state
- Health monitoring and auto-restart

## Message Handling
- Parse commands with prefix or slash command patterns
- Handle rich message formats (embeds, buttons, modals)
- Implement conversation flows for multi-step interactions
- Process file attachments and media
- Handle mentions and direct messages

## Platform Integration
- OAuth2 for user authentication
- Webhook receivers for real-time events
- API calls for channel and user management
- Interactive components (buttons, menus, forms)
- Scheduled messages and reminders

## Best Practices
- Implement proper error handling and user feedback
- Rate limit API calls to stay within platform limits
- Log all interactions for debugging
- Use environment variables for secrets
- Implement graceful shutdown and reconnection
