---
name: github_copilot
description: "GitHub Copilot integration and best practices"
version: "1.0"
trigger: "github copilot ai pair programming"
platforms: []
requires_tools: ["run_command"]
---

# GitHub Copilot

## Purpose
Maximize productivity with GitHub Copilot through effective prompting, context management, and workflow integration.

## Instructions
1. Set up Copilot in your IDE
2. Use clear comments and function signatures to guide suggestions
3. Review and validate all suggestions before accepting
4. Use inline chat for complex tasks
5. Leverage workspace context for better suggestions

## Effective Prompting
- Write descriptive function names and comments
- Provide type annotations for better suggestions
- Use doc comments to describe expected behavior
- Break complex tasks into smaller functions
- Include example inputs/outputs in comments

## Best Practices
- Always review suggestions for correctness and security
- Use Copilot for boilerplate, not critical logic
- Verify generated code passes tests
- Do not accept suggestions containing hardcoded secrets
- Keep your codebase well-organized for better context
