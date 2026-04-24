---
name: load_testing
description: "Load testing and performance benchmarking"
version: "1.0"
trigger: "load testing k6 jmeter gatling"
platforms: []
requires_tools: ["run_command"]
---

# Load testing and performance benchmarking

## Purpose
Load testing and performance benchmarking for reliable, scalable, and maintainable infrastructure and deployment pipelines.

## Instructions
1. Assess the current infrastructure and requirements
2. Design the solution following infrastructure-as-code principles
3. Implement with proper version control and review processes
4. Test in staging environment before production deployment
5. Document runbooks and operational procedures

## Core Principles
- Infrastructure as Code: all configuration in version control
- Immutable infrastructure: replace, don't modify
- Automation: minimize manual intervention
- Monitoring: observe everything, alert on what matters
- Resilience: plan for failure, practice recovery

## Implementation Steps
1. Define requirements and constraints
2. Choose appropriate tools and services
3. Write infrastructure code with proper abstraction
4. Implement CI/CD pipeline for infrastructure changes
5. Set up monitoring, alerting, and logging
6. Create runbooks for common operations
7. Test disaster recovery procedures

## Security Considerations
- Principle of least privilege for all access
- Encrypt data in transit and at rest
- Rotate credentials regularly
- Audit access and changes
- Scan infrastructure for misconfigurations

## Operational Excellence
- Maintain documentation alongside code
- Conduct regular disaster recovery drills
- Review and update runbooks after incidents
- Track MTTR, change failure rate, deployment frequency
- Implement progressive rollouts to limit blast radius

## Best Practices
- Use modules and reusable components
- Tag all resources for cost allocation and management
- Implement proper state management and locking
- Use blue-green or canary deployments for zero-downtime updates
- Automate routine maintenance tasks
