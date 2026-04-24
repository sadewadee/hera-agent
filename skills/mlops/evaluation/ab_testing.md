---
name: ab_testing
description: "A/B testing for ML model evaluation in production"
version: "1.0"
trigger: "ab testing model evaluation production"
platforms: []
requires_tools: ["run_command"]
---

# A/B Testing for ML Models

## Purpose
Design and execute A/B tests to compare ML model versions in production, with statistical rigor and proper traffic allocation.

## Instructions
1. Define the hypothesis and success metrics (conversion rate, latency, error rate)
2. Calculate required sample size for statistical significance
3. Set up traffic splitting between control (current model) and treatment (new model)
4. Monitor metrics during the test period
5. Analyze results with proper statistical tests and make a go/no-go decision

## Sample Size Calculation
- Use power analysis to determine minimum sample size
- Typical parameters: 80% power, 5% significance level, minimum detectable effect
- Account for multiple comparisons if testing more than one variant
- Consider practical significance vs statistical significance

## Traffic Splitting Strategies
- **Random split**: Hash user ID to deterministically assign users to groups
- **Ramped rollout**: Start at 1%, increase to 5%, 10%, 50% as confidence grows
- **Shadow mode**: Route all traffic to both models, only serve control responses
- **Interleaving**: Mix results from both models in a single response (for ranking)

## Statistical Analysis
- Use two-sample t-test for continuous metrics (latency, score)
- Use chi-squared test for binary metrics (click-through, conversion)
- Apply Bonferroni correction for multiple hypothesis testing
- Check for novelty effects by analyzing metrics over time windows
- Use sequential testing (SPRT) for early stopping decisions

## Guardrail Metrics
- Latency p99 must not regress by more than 10%
- Error rate must not increase by more than 0.1%
- Revenue metrics must not decrease by more than 1%
- User satisfaction scores must remain stable

## Common Pitfalls
- Ending tests too early based on initial trends
- Not accounting for day-of-week or seasonal effects
- Ignoring interaction effects between simultaneous experiments
- Using the wrong statistical test for the metric type
- Contaminating groups by allowing users to switch between variants
