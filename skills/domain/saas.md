---
name: saas
description: "SaaS product development patterns"
version: "1.0"
trigger: "saas subscription multi-tenant product"
platforms: []
requires_tools: []
---

# SaaS Development

## Purpose
Design and build SaaS products with multi-tenancy, subscription billing, and growth-oriented architecture.

## Instructions
1. Define the product and target market
2. Design multi-tenant architecture
3. Implement subscription and billing
4. Build onboarding and activation flows
5. Monitor key SaaS metrics

## Multi-Tenancy
- **Shared database**: One database, tenant_id column
- **Schema per tenant**: Separate schemas, shared database
- **Database per tenant**: Full isolation, highest cost
- Choose based on isolation, cost, and compliance needs

## Key Metrics
- MRR/ARR (Monthly/Annual Recurring Revenue)
- Churn rate (monthly and annual)
- Customer Acquisition Cost (CAC)
- Lifetime Value (LTV)
- Net Revenue Retention (NRR)

## Best Practices
- Design for multi-tenancy from day one
- Implement proper data isolation and access control
- Build self-service onboarding
- Monitor usage patterns for product insights
- Plan billing and subscription management early
