---
name: hypothesis_testing
description: "Statistical hypothesis testing for ML research"
version: "1.0"
trigger: "hypothesis testing statistical significance p-value"
platforms: []
requires_tools: ["run_command"]
---

# Statistical Hypothesis Testing for ML

## Purpose
Apply rigorous statistical methods to validate ML research claims, ensuring that reported improvements are real and not artifacts of random variation.

## Instructions
1. Formulate null and alternative hypotheses
2. Select appropriate statistical test for the comparison
3. Determine required sample size for desired power
4. Run experiments with proper controls and randomization
5. Report results with confidence intervals, not just p-values

## Common Tests for ML
| Comparison | Test | When to Use |
|------------|------|-------------|
| Two models on one dataset | Paired t-test or Wilcoxon signed-rank | Comparing accuracy/F1/etc. |
| Two models on multiple datasets | Wilcoxon signed-rank test | Cross-dataset comparison |
| Multiple models | Friedman test + Nemenyi post-hoc | Model selection |
| Training run variance | Bootstrap confidence intervals | Single dataset |

## Proper ML Comparison Protocol
1. Fix all random seeds and run multiple seeds (5-10 minimum)
2. Use the same train/val/test splits across models
3. Report mean and standard deviation, not best run
4. Use paired tests (same data splits) for tighter bounds
5. Apply Bonferroni correction for multiple comparisons

## Bootstrap Confidence Intervals
```python
import numpy as np

def bootstrap_ci(scores, n_bootstrap=10000, confidence=0.95):
    means = []
    for _ in range(n_bootstrap):
        sample = np.random.choice(scores, size=len(scores), replace=True)
        means.append(np.mean(sample))
    lower = np.percentile(means, (1 - confidence) / 2 * 100)
    upper = np.percentile(means, (1 + confidence) / 2 * 100)
    return lower, upper
```

## Common Pitfalls
- Claiming significance from a single train/test split
- Not accounting for hyperparameter tuning on test set
- Reporting best of N runs instead of average
- Using parametric tests on non-normal distributions
- P-hacking by trying multiple tests and reporting the one that works
