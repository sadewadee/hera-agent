---
name: mixed_precision
description: "Mixed precision training for faster model training"
version: "1.0"
trigger: "mixed precision fp16 bf16 training"
platforms: []
requires_tools: ["run_command"]
---

# Mixed Precision Training

## Purpose
Speed up model training by 2-3x and reduce memory usage by using lower-precision floating-point formats (FP16/BF16) for most operations while maintaining model quality.

## Instructions
1. Verify GPU supports mixed precision (V100+, A100, H100, or AMD MI250+)
2. Enable automatic mixed precision in your training framework
3. Configure loss scaling to prevent gradient underflow
4. Monitor training stability (loss spikes, NaN gradients)
5. Validate final model accuracy matches FP32 baseline

## Precision Formats
| Format | Bits | Range | Precision | Best For |
|--------|------|-------|-----------|----------|
| FP32   | 32   | Large | High      | Baseline, accumulation |
| FP16   | 16   | Small | Medium    | Forward/backward pass (with loss scaling) |
| BF16   | 16   | Large | Lower     | Forward/backward pass (no loss scaling needed) |
| TF32   | 19   | Large | Medium    | A100+ internal computation |

## PyTorch Implementation
```python
from torch.cuda.amp import autocast, GradScaler

scaler = GradScaler()

for batch in dataloader:
    optimizer.zero_grad()
    with autocast(dtype=torch.float16):
        output = model(batch)
        loss = criterion(output, targets)
    scaler.scale(loss).backward()
    scaler.step(optimizer)
    scaler.update()
```

## BF16 (Recommended for A100+)
```python
# BF16 doesn't need loss scaling
with autocast(dtype=torch.bfloat16):
    output = model(batch)
    loss = criterion(output, targets)
loss.backward()
optimizer.step()
```

## Troubleshooting
- **NaN loss**: Reduce learning rate, check for division by zero, increase loss scale
- **Divergence**: Use BF16 instead of FP16, or reduce gradient clipping threshold
- **No speedup**: Batch size may be too small to saturate GPU tensor cores
- **Memory not reduced**: Optimizer states still in FP32 (expected, use ZeRO for those)

## Best Practices
- Use BF16 on Ampere+ GPUs (A100, H100) - simpler and more stable than FP16
- Keep master weights and optimizer state in FP32
- Use gradient clipping (max_norm=1.0) as additional stability measure
- Loss scaling is automatic with GradScaler - only tune if issues arise
