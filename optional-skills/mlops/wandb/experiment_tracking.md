---
name: wandb_tracking
description: "Experiment tracking with Weights & Biases"
version: "1.0"
trigger: "wandb weights biases experiment tracking"
platforms: []
requires_tools: ["run_command"]
---

# Weights & Biases Experiment Tracking

## Purpose
Track ML experiments, hyperparameters, metrics, and artifacts using Weights & Biases for reproducibility and collaboration.

## Instructions
1. Initialize a W&B run with project and config
2. Log metrics during training
3. Log artifacts (models, datasets, predictions)
4. Compare runs in the W&B dashboard
5. Use sweeps for hyperparameter optimization

## Basic Usage
```python
import wandb

wandb.init(
    project="my-ml-project",
    config={
        "learning_rate": 2e-5,
        "batch_size": 32,
        "epochs": 10,
        "model": "bert-base",
    },
)

for epoch in range(config.epochs):
    train_loss = train_one_epoch()
    val_loss, val_acc = evaluate()
    wandb.log({
        "train/loss": train_loss,
        "val/loss": val_loss,
        "val/accuracy": val_acc,
        "epoch": epoch,
    })

wandb.finish()
```

## Artifact Tracking
```python
artifact = wandb.Artifact("model-v1", type="model")
artifact.add_file("model.pt")
wandb.log_artifact(artifact)

# Dataset versioning
data_artifact = wandb.Artifact("training-data-v2", type="dataset")
data_artifact.add_dir("data/processed/")
wandb.log_artifact(data_artifact)
```

## Hyperparameter Sweeps
```yaml
# sweep.yaml
method: bayes
metric:
  name: val/accuracy
  goal: maximize
parameters:
  learning_rate:
    min: 1e-6
    max: 1e-3
  batch_size:
    values: [16, 32, 64]
  dropout:
    min: 0.0
    max: 0.5
```

## Integration with Frameworks
- HuggingFace Trainer: `report_to="wandb"` in TrainingArguments
- PyTorch Lightning: `WandbLogger` in Trainer
- Keras: `WandbCallback` in model.fit()

## Best Practices
- Always log config for reproducibility
- Use run groups to organize related experiments
- Tag runs with meaningful labels (baseline, experiment-v2)
- Log sample predictions for qualitative analysis
- Use alerts for long-running training jobs
