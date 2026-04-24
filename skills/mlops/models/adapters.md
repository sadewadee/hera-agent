---
name: adapters
description: "Parameter-efficient fine-tuning with adapters"
version: "1.0"
trigger: "adapters peft parameter efficient fine-tuning"
platforms: []
requires_tools: ["run_command"]
---

# Adapter-Based Fine-Tuning

## Purpose
Fine-tune large models efficiently by training small adapter modules instead of all parameters, reducing compute and storage requirements by 10-100x.

## Instructions
1. Select the base model and target task
2. Choose an adapter method (LoRA, prefix tuning, IA3, etc.)
3. Configure adapter hyperparameters (rank, target modules, alpha)
4. Train adapters on task-specific data
5. Merge adapters with base model or serve separately

## Adapter Methods
| Method | Parameters | Quality | Speed | Use Case |
|--------|-----------|---------|-------|----------|
| LoRA   | 0.1-1%    | High    | Fast  | General fine-tuning |
| QLoRA  | 0.1-1%    | High    | Fast  | Memory-constrained |
| Prefix Tuning | <0.1% | Medium | Fast | Text generation |
| IA3    | <0.01%    | Medium  | Fastest | Few-shot adaptation |
| AdaLoRA | 0.1-1%   | High    | Medium | Automatic rank selection |

## LoRA Configuration
```python
from peft import LoraConfig, get_peft_model

config = LoraConfig(
    r=16,                     # rank (4-64, higher = more capacity)
    lora_alpha=32,            # scaling factor (usually 2x rank)
    target_modules=["q_proj", "v_proj", "k_proj", "o_proj"],
    lora_dropout=0.05,
    bias="none",
    task_type="CAUSAL_LM",
)
model = get_peft_model(base_model, config)
```

## Multi-Adapter Serving
- Store adapters separately from base model (small files, easy to swap)
- Load multiple adapters on a single base model instance
- Route requests to appropriate adapter based on task/user
- Hot-swap adapters without restarting the server

## Best Practices
- Start with rank=16 and increase if underfitting
- Target attention projection layers (q, k, v, o) for best results
- Use QLoRA for models that don't fit in GPU memory at full precision
- Merge adapters into base model for inference speed (no adapter overhead)
- Version control adapter weights separately from base model
