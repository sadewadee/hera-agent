---
title: Email
sidebar_label: Email
---

# Email Integration

Send emails to Hera and receive replies via IMAP/SMTP. Works with Gmail, Outlook, Fastmail, Proton Mail, and any standard email provider.

## How It Works

Hera polls your inbox via IMAP for new messages, processes them with the agent, and sends replies via SMTP. Each email thread maintains its own session context.

## Prerequisites

- An email account dedicated to the bot
- IMAP and SMTP access enabled (may require an app password for Gmail/Outlook)

## Gmail Setup

1. Enable 2-factor authentication on your Google account
2. Go to **Google Account → Security → App passwords**
3. Create an app password for "Mail" on "Other device" (name it "Hera")
4. Use this app password as `EMAIL_PASSWORD`

## Configuration

### Environment Variables

```bash
EMAIL_IMAP_HOST=imap.gmail.com
EMAIL_IMAP_PORT=993
EMAIL_SMTP_HOST=smtp.gmail.com
EMAIL_SMTP_PORT=587
EMAIL_USERNAME=yourbot@gmail.com
EMAIL_PASSWORD=your-app-password
EMAIL_FROM_ADDRESS=yourbot@gmail.com
EMAIL_POLL_INTERVAL=60   # seconds between inbox checks
EMAIL_ALLOW_LIST=alice@example.com,bob@example.com
```

### Config File (`~/.hera/config.yaml`)

```yaml
gateway:
  platforms:
    email:
      enabled: true
      extra:
        imap_host: imap.gmail.com
        imap_port: "993"
        smtp_host: smtp.gmail.com
        smtp_port: "587"
        username: yourbot@gmail.com
        password: ${EMAIL_PASSWORD}
        from_address: yourbot@gmail.com
        poll_interval: "60"
      allow_list:
        - alice@example.com
```

## Starting Hera with Email

```bash
hera gateway --platform email
```

## Common IMAP/SMTP Settings

| Provider | IMAP Host | IMAP Port | SMTP Host | SMTP Port |
|----------|-----------|-----------|-----------|-----------|
| Gmail | imap.gmail.com | 993 | smtp.gmail.com | 587 |
| Outlook | outlook.office365.com | 993 | smtp-mail.outlook.com | 587 |
| Fastmail | imap.fastmail.com | 993 | smtp.fastmail.com | 587 |
| Yahoo | imap.mail.yahoo.com | 993 | smtp.mail.yahoo.com | 587 |

## Security

- Use an app password rather than your main account password
- Set `allow_list` to restrict which email addresses can interact with the agent
- Use a dedicated email account for the bot, not your personal account
- IMAP connections use TLS by default on port 993
