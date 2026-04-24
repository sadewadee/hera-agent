---
name: inference_cost
description: "Inference cost estimation and optimization"
version: "1.0"
trigger: "inference cost pricing optimization budget"
platforms: []
requires_tools: ["run_command"]
---

# Inference Cost Optimization

## Purpose
Estimate and optimize ML inference costs across different deployment options.

## Instructions
1. Calculate current inference costs (compute, bandwidth, storage)
2. Benchmark different deployment options
3. Identify cost optimization opportunities
4. Implement cost-saving measures
5. Monitor and track savings

## Cost Factors
- GPU/CPU instance hours
- Data transfer costs
- Storage for model artifacts
- Request volume and batching efficiency
- Cold start frequency

## Optimization Strategies
- Use spot/preemptible instances for non-critical workloads
- Implement request batching to maximize GPU utilization
- Use model quantization to reduce compute requirements
- Auto-scale to zero during idle periods
- Choose the right GPU type for your model size

## Best Practices
- Benchmark cost per prediction across providers
- Set up billing alerts and budgets
- Monitor GPU utilization (target >80%)
- Use caching for repeated queries
- Regularly review and right-size instances
