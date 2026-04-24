---
name: curriculum
description: "Curriculum learning strategies for model training"
version: "1.0"
trigger: "curriculum learning training strategy"
platforms: []
requires_tools: ["run_command"]
---

# Curriculum Learning

## Purpose
Improve training efficiency and model quality by presenting training examples in a meaningful order, progressing from easy to hard or simple to complex.

## Instructions
1. Define difficulty metrics for your training data
2. Sort or bin training examples by difficulty
3. Design the curriculum schedule (pacing function)
4. Train the model following the curriculum
5. Evaluate against random-order baseline

## Difficulty Metrics
- **Loss-based**: Use a pre-trained model's loss on each example
- **Length-based**: Shorter sequences or smaller images first
- **Complexity-based**: Vocabulary diversity, syntactic depth
- **Noise-based**: Clean examples first, noisy examples later
- **Confidence-based**: High-confidence examples first

## Curriculum Strategies
- **Linear**: Gradually increase data pool from easy to all
- **Exponential**: Double available data at each step
- **Self-paced**: Model selects examples it can learn from (based on current loss)
- **Anti-curriculum**: Hard examples first (useful for some domains)
- **Competence-based**: Unlock harder data when model achieves competence threshold

## Implementation Pattern
```python
# Sort by difficulty
difficulties = compute_difficulty(dataset)
sorted_indices = np.argsort(difficulties)

# Curriculum pacing: linear increase
for epoch in range(num_epochs):
    fraction = min(1.0, 0.2 + 0.8 * epoch / num_epochs)
    n_samples = int(fraction * len(dataset))
    curriculum_indices = sorted_indices[:n_samples]
    train_epoch(dataset[curriculum_indices])
```

## Benefits
- Faster convergence (10-30% fewer steps to same accuracy)
- Better generalization on hard examples
- More stable training (fewer loss spikes)
- Can reduce total training compute

## When to Use
- Large, heterogeneous datasets
- Tasks with clear easy/hard distinction
- Training instability with random ordering
- Limited compute budget (get more from less training)
