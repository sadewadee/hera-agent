---
name: owasp
description: "OWASP security guidelines and testing"
version: "1.0"
trigger: "owasp security web application testing"
platforms: []
requires_tools: ["run_command"]
---

# OWASP Security

## Purpose
Apply OWASP guidelines for web application security testing and secure development practices.

## Instructions
1. Review OWASP Top 10 against your application
2. Run automated scanning tools
3. Perform manual testing for business logic flaws
4. Document findings with OWASP references
5. Implement mitigations following OWASP guides

## OWASP Top 10 (2021)
1. Broken Access Control
2. Cryptographic Failures
3. Injection
4. Insecure Design
5. Security Misconfiguration
6. Vulnerable and Outdated Components
7. Identification and Authentication Failures
8. Software and Data Integrity Failures
9. Security Logging and Monitoring Failures
10. Server-Side Request Forgery (SSRF)

## Testing Tools
- OWASP ZAP: Free web app security scanner
- Burp Suite: Professional web security testing
- OWASP Dependency-Check: Known vulnerability scanning
- OWASP ASVS: Application Security Verification Standard

## Best Practices
- Use OWASP ASVS as your security requirements baseline
- Integrate security testing into CI/CD pipeline
- Train developers on OWASP Top 10
- Regular security reviews using OWASP Testing Guide
- Use OWASP Cheat Sheets for implementation guidance
