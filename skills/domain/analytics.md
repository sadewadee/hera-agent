---
name: analytics
description: "Business analytics and reporting"
version: "1.0"
trigger: "analytics business reporting dashboard kpi"
platforms: []
requires_tools: ["run_command"]
---

# Business Analytics

## Purpose
Build analytics dashboards and reports for data-driven business decisions.

## Instructions
1. Define business questions and KPIs
2. Identify data sources and collection methods
3. Build data pipelines and transformations
4. Create dashboards and reports
5. Set up alerts for key metric changes

## Analytics Stack
- Data warehouse: BigQuery, Snowflake, Redshift
- ETL/ELT: dbt, Fivetran, Airbyte
- BI tools: Metabase, Looker, Superset, Tableau
- Metrics: SQL-based metric layers
- Alerting: based on threshold triggers

## Key Metrics
- Revenue metrics: MRR, ARR, ARPU
- Growth metrics: new users, activation, retention
- Product metrics: DAU/MAU, feature usage, engagement
- Operational metrics: response time, error rate, uptime

## Best Practices
- Define metrics clearly with documentation
- Use version-controlled SQL/dbt models
- Validate data quality with automated tests
- Make dashboards self-service for stakeholders
- Review metrics regularly and update as needed
