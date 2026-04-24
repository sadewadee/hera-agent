---
name: email_security
description: "Email security and phishing protection"
version: "1.0"
trigger: "email security phishing protection spam"
platforms: []
requires_tools: ["run_command"]
---

# Email Security

## Purpose
Protect email communications from phishing, spam, and spoofing attacks.

## Instructions
1. Configure email authentication (SPF, DKIM, DMARC)
2. Set up anti-spam and anti-phishing filters
3. Train users to recognize phishing attempts
4. Monitor and respond to security incidents
5. Review and update policies regularly

## Authentication
- **SPF**: Specify authorized sending servers
- **DKIM**: Cryptographically sign outgoing emails
- **DMARC**: Policy for handling authentication failures

## Best Practices
- Implement DMARC with at least `p=quarantine`
- Monitor DMARC reports for spoofing attempts
- Train employees on phishing recognition
- Use email encryption for sensitive content
- Regularly audit authorized senders list
