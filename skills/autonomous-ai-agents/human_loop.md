---
name: human_loop
description: "Human-in-the-loop agent designs"
version: "1.0"
trigger: "human loop agent approval review"
platforms: []
requires_tools: []
---

# Human-in-the-loop agent designs

## Purpose
Human-in-the-loop agent designs for building reliable and controllable autonomous agent systems.

## Instructions
1. Define the agent interaction pattern
2. Design approval and review workflows
3. Implement the agent pipeline with control points
4. Test edge cases and failure modes
5. Monitor agent behavior in production

## Architecture
- Define clear boundaries between autonomous and supervised operations
- Implement checkpoints for human review
- Design escalation paths for uncertain decisions
- Log all agent decisions for audit
- Set up monitoring for anomalous behavior

## Best Practices
- Start with more human oversight, reduce as trust builds
- Define clear criteria for autonomous vs supervised decisions
- Implement comprehensive logging for auditability
- Test with diverse and adversarial inputs
- Regularly review agent decisions for quality
