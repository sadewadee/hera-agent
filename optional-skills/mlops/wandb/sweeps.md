---
name: wandb_sweeps
description: "Hyperparameter optimization with W&B Sweeps"
version: "1.0"
trigger: "wandb sweeps hyperparameter optimization"
platforms: []
requires_tools: ["run_command"]
---

# W&B Sweeps

## Purpose
Run distributed hyperparameter searches using W&B Sweeps with Bayesian optimization, grid search, or random search.

## Instructions
1. Define search space and optimization strategy
2. Create sweep configuration
3. Initialize sweep and launch agents
4. Monitor runs in the sweep dashboard
5. Analyze results and select best configuration

## Search Methods
| Method | When to Use | Efficiency |
|--------|-------------|------------|
| Bayesian (bayes) | Default, most scenarios | High |
| Random | Large search spaces, unknown landscape | Medium |
| Grid | Small discrete spaces, exhaustive search | Low |

## Configuration
```python
sweep_config = {
    "method": "bayes",
    "metric": {"name": "val_loss", "goal": "minimize"},
    "parameters": {
        "learning_rate": {"distribution": "log_uniform_values", "min": 1e-6, "max": 1e-3},
        "batch_size": {"values": [8, 16, 32, 64]},
        "weight_decay": {"distribution": "uniform", "min": 0, "max": 0.1},
        "warmup_steps": {"distribution": "int_uniform", "min": 0, "max": 500},
    },
    "early_terminate": {
        "type": "hyperband",
        "min_iter": 3,
        "eta": 3,
    },
}

sweep_id = wandb.sweep(sweep_config, project="my-project")
wandb.agent(sweep_id, function=train, count=50)
```

## Parallel Agents
```bash
# Launch multiple agents on different machines/GPUs
wandb agent username/project/sweep_id
```

## Early Termination
- **Hyperband**: Stop underperforming runs based on learning curve
- Reduces total compute by 3-5x
- Set `min_iter` to minimum epochs before termination is considered

## Best Practices
- Start with random search to understand the landscape
- Switch to Bayesian for focused optimization
- Log all relevant metrics (not just the optimization target)
- Use early termination to avoid wasting compute on bad configs
- Run at least 20-50 trials for Bayesian to be effective
