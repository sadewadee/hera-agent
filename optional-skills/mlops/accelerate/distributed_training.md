---
name: accelerate_distributed
description: "Distributed training with HuggingFace Accelerate"
version: "1.0"
trigger: "accelerate distributed training multi-gpu"
platforms: []
requires_tools: ["run_command"]
---

# HuggingFace Accelerate

## Purpose
Simplify distributed training across multiple GPUs and nodes using HuggingFace Accelerate with minimal code changes.

## Instructions
1. Install accelerate and run `accelerate config` to set up your environment
2. Wrap your training code with Accelerate's primitives
3. Launch with `accelerate launch` instead of `python`
4. Monitor distributed training metrics
5. Handle checkpointing and evaluation in distributed context

## Minimal Code Changes
```python
from accelerate import Accelerator

accelerator = Accelerator()

# Wrap model, optimizer, dataloader
model, optimizer, dataloader = accelerator.prepare(model, optimizer, dataloader)

for batch in dataloader:
    outputs = model(**batch)
    loss = outputs.loss
    accelerator.backward(loss)  # replaces loss.backward()
    optimizer.step()
    optimizer.zero_grad()
```

## Launch Commands
```bash
# Single machine, multi-GPU
accelerate launch --num_processes 4 train.py

# Multi-node
accelerate launch --num_machines 2 --machine_rank 0 --main_process_ip 10.0.0.1 train.py

# With DeepSpeed
accelerate launch --use_deepspeed --deepspeed_config ds_config.json train.py
```

## Mixed Precision
```python
accelerator = Accelerator(mixed_precision="bf16")  # or "fp16"
```

## Gradient Accumulation
```python
accelerator = Accelerator(gradient_accumulation_steps=4)

for batch in dataloader:
    with accelerator.accumulate(model):
        outputs = model(**batch)
        loss = outputs.loss
        accelerator.backward(loss)
        optimizer.step()
        optimizer.zero_grad()
```

## Best Practices
- Use `accelerator.is_main_process` for logging and saving
- Use `accelerator.gather()` to collect metrics from all processes
- Save checkpoints with `accelerator.save_state()`
- Set `NCCL_DEBUG=INFO` for debugging communication issues
