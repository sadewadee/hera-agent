---
name: fly_io
description: "Fly.io application deployment"
version: "1.0"
trigger: "fly.io flyctl deploy edge"
platforms: []
requires_tools: ["run_command"]
---

# Fly.io Deployment

## Purpose
Deploy applications globally on Fly.io's edge infrastructure with minimal configuration.

## Instructions
1. Install flyctl CLI
2. Create and configure Fly app
3. Configure scaling and regions
4. Deploy application
5. Monitor and manage

## Quick Deploy
```bash
flyctl launch
flyctl deploy
```

## Configuration (fly.toml)
```toml
app = "my-app"
primary_region = "ord"

[http_service]
  internal_port = 8080
  force_https = true

[build]
  [build.args]
    GO_VERSION = "1.22"

[[vm]]
  size = "shared-cpu-1x"
  memory = "256mb"
```

## Best Practices
- Deploy to multiple regions for low latency
- Use Fly volumes for persistent storage
- Configure health checks for reliability
- Use secrets for sensitive configuration
- Monitor with Fly metrics and dashboards
