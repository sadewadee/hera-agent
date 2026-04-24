---
name: pruning
description: "Neural network pruning for model compression"
version: "1.0"
trigger: "pruning model compression sparsity"
platforms: []
requires_tools: ["run_command"]
---

# Model Pruning

## Purpose
Reduce model size and computational cost by removing redundant parameters while maintaining acceptable accuracy.

## Instructions
1. Train the full model to convergence
2. Analyze parameter importance using selected criteria
3. Apply pruning (structured or unstructured) at target sparsity
4. Fine-tune the pruned model to recover accuracy
5. Repeat pruning and fine-tuning iteratively if needed

## Pruning Types
- **Unstructured pruning**: Zero out individual weights (high sparsity, needs sparse hardware)
- **Structured pruning**: Remove entire neurons, channels, or attention heads (hardware-friendly)
- **Semi-structured (N:M)**: N zeros in every M consecutive weights (NVIDIA A100+ support)

## Pruning Criteria
- **Magnitude-based**: Remove weights with smallest absolute values
- **Gradient-based**: Remove weights with smallest gradient magnitudes
- **Taylor expansion**: Estimate importance based on loss change when weight is removed
- **Movement pruning**: Track weight movement during fine-tuning, prune non-moving weights

## Iterative Pruning Schedule
1. Train to convergence
2. Prune 20% of remaining weights
3. Fine-tune for 10-20% of original training steps
4. Repeat until target sparsity reached
5. Final fine-tuning at target sparsity

## Best Practices
- Start with structured pruning if deploying on standard hardware
- Use gradual pruning schedules (cubic sparsity ramp) over abrupt pruning
- Monitor per-layer sparsity to avoid over-pruning critical layers
- Combine with quantization for maximum compression
- Validate on downstream tasks, not just training metrics
