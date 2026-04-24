---
name: pytorch_lightning
description: "Structured training with PyTorch Lightning"
version: "1.0"
trigger: "pytorch lightning training framework"
platforms: []
requires_tools: ["run_command"]
---

# PyTorch Lightning

## Purpose
Structure PyTorch training code using Lightning for reproducibility, multi-GPU support, and reduced boilerplate.

## Instructions
1. Define a LightningModule with training_step, configure_optimizers
2. Create a LightningDataModule for data loading
3. Configure a Trainer with callbacks and logging
4. Train with built-in distributed, mixed precision, and checkpointing
5. Use callbacks for custom training behaviors

## LightningModule
```python
import pytorch_lightning as pl

class TextClassifier(pl.LightningModule):
    def __init__(self, model_name, num_labels, lr=2e-5):
        super().__init__()
        self.save_hyperparameters()
        self.model = AutoModelForSequenceClassification.from_pretrained(
            model_name, num_labels=num_labels
        )

    def training_step(self, batch, batch_idx):
        outputs = self.model(**batch)
        self.log("train_loss", outputs.loss, prog_bar=True)
        return outputs.loss

    def validation_step(self, batch, batch_idx):
        outputs = self.model(**batch)
        self.log("val_loss", outputs.loss, prog_bar=True)

    def configure_optimizers(self):
        return torch.optim.AdamW(self.parameters(), lr=self.hparams.lr)
```

## Trainer Configuration
```python
trainer = pl.Trainer(
    max_epochs=10,
    accelerator="gpu",
    devices=4,
    strategy="ddp",
    precision="bf16-mixed",
    gradient_clip_val=1.0,
    accumulate_grad_batches=4,
    callbacks=[
        pl.callbacks.ModelCheckpoint(monitor="val_loss", mode="min"),
        pl.callbacks.EarlyStopping(monitor="val_loss", patience=3),
        pl.callbacks.LearningRateMonitor(),
    ],
    logger=pl.loggers.WandbLogger(project="my-project"),
)
trainer.fit(model, datamodule=dm)
```

## Key Callbacks
- `ModelCheckpoint`: Save best and last models
- `EarlyStopping`: Stop training when metric plateaus
- `LearningRateMonitor`: Log LR schedule
- `GradientAccumulationScheduler`: Dynamic accumulation
- `StochasticWeightAveraging`: Improve generalization

## Best Practices
- Use `save_hyperparameters()` for automatic config saving
- Use `self.log()` for all metrics (auto-handles distributed reduction)
- Use LightningDataModule to encapsulate data logic
- Test with `Trainer(fast_dev_run=True)` before full training
- Use `trainer.predict()` for inference with same distributed setup
