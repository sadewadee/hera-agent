---
name: secret_scanning
description: "Automated secret detection and scanning"
version: "1.0"
trigger: "secret scanning detection credential leak"
platforms: []
requires_tools: ["run_command"]
---

# Secret Scanning

## Purpose
Detect and prevent accidental exposure of secrets (API keys, passwords, tokens) in code repositories.

## Instructions
1. Set up pre-commit hooks for local scanning
2. Configure CI/CD pipeline scanning
3. Scan repository history for past leaks
4. Implement remediation workflow for detected secrets
5. Monitor and alert on new detections

## Tools
- **trufflehog**: Scans Git history for high-entropy strings and known patterns
- **gitleaks**: Fast scanning with custom rules
- **detect-secrets**: Yelp's tool with baseline support
- **GitHub Secret Scanning**: Built-in for GitHub repos

## Pre-commit Hook
```yaml
# .pre-commit-config.yaml
repos:
  - repo: https://github.com/gitleaks/gitleaks
    rev: v8.18.0
    hooks:
      - id: gitleaks
```

## Remediation
1. Immediately revoke the exposed secret
2. Generate a new secret
3. Update all systems using the old secret
4. Remove the secret from git history if possible
5. Add patterns to prevent future leaks

## Best Practices
- Scan on every commit (pre-commit hook)
- Scan in CI/CD as a safety net
- Maintain an allowlist for false positives
- Train developers on secret management
- Use secret managers (Vault, AWS Secrets Manager) instead of env files
