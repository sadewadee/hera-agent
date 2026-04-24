---
name: db_migration
description: "Database migration strategies"
version: "1.0"
trigger: "database migration schema version"
platforms: []
requires_tools: ["run_command"]
---

# Database migration strategies

## Purpose
Database migration strategies for safely moving systems, data, and infrastructure with minimal downtime.

## Instructions
1. Assess the current state and migration requirements
2. Design the migration plan with rollback strategy
3. Test the migration in a staging environment
4. Execute the migration with monitoring
5. Verify success and clean up old resources

## Planning
- Inventory all components to migrate
- Identify dependencies and ordering constraints
- Define success criteria and rollback triggers
- Estimate downtime and communicate to stakeholders
- Prepare runbooks for each migration step

## Execution
- Follow the tested migration plan exactly
- Monitor key metrics during migration
- Communicate progress to stakeholders
- Be ready to execute rollback at any step
- Verify each step before proceeding

## Validation
- Run automated tests against migrated system
- Compare outputs between old and new systems
- Check data integrity and completeness
- Verify performance meets requirements
- Monitor for errors in the days following migration

## Best Practices
- Test migrations in staging with production-like data
- Always have a rollback plan for every step
- Minimize the migration window
- Communicate clearly with all stakeholders
- Document the migration for future reference
