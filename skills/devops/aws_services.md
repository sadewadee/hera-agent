---
name: aws_services
description: "AWS service configuration and management"
version: "1.0"
trigger: "aws cloud services ec2 s3 lambda"
platforms: []
requires_tools: ["run_command"]
---

# AWS service configuration and management

## Purpose
AWS service configuration and management for cloud infrastructure management.

## Instructions
1. Assess infrastructure requirements
2. Select appropriate cloud services
3. Configure with infrastructure as code
4. Deploy with proper security and monitoring
5. Optimize for cost and performance

## Service Selection
- Compute: VMs, containers, serverless
- Storage: Object, block, file systems
- Databases: SQL, NoSQL, cache
- Networking: VPC, load balancers, CDN
- Security: IAM, KMS, WAF

## Best Practices
- Use infrastructure as code (Terraform, CDK, Pulumi)
- Implement least privilege access
- Enable logging and monitoring for all services
- Set up billing alerts and cost optimization
- Use multi-AZ/region for high availability
