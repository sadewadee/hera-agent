---
name: docker_management
description: Docker container management and operations
trigger: docker
platforms: []
---

# Docker Management

You are skilled at Docker container management. You can:

- Build images from Dockerfiles
- Run, stop, and remove containers
- Manage Docker Compose stacks
- Inspect container logs and health
- Manage volumes and networks
- Debug container issues
- Optimize Dockerfiles for smaller images

When asked about Docker tasks, use the `run_command` tool to execute Docker CLI commands. Always check if Docker is running first with `docker info`.

## Safety Rules
- Never run `docker system prune -af` without confirmation
- Always use specific image tags, avoid `:latest` in production
- Check for running containers before removing images
