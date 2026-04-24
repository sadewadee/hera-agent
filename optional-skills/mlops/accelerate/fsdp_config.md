---
name: fsdp_config
description: "Configure FSDP with HuggingFace Accelerate"
version: "1.0"
trigger: "fsdp accelerate sharding config"
platforms: []
requires_tools: ["run_command"]
---

# FSDP with Accelerate

## Purpose
Configure Fully Sharded Data Parallel (FSDP) through HuggingFace Accelerate for training models too large to fit on a single GPU.

## Instructions
1. Configure FSDP settings in `accelerate config` or YAML file
2. Choose sharding strategy based on model size and GPU count
3. Set auto-wrapping policy for transformer layers
4. Configure mixed precision and activation checkpointing
5. Handle checkpointing with proper FSDP state dict options

## Configuration YAML
```yaml
compute_environment: LOCAL_MACHINE
distributed_type: FSDP
fsdp_config:
  fsdp_auto_wrap_policy: TRANSFORMER_BASED_WRAP
  fsdp_backward_prefetch: BACKWARD_PRE
  fsdp_sharding_strategy: FULL_SHARD
  fsdp_state_dict_type: SHARDED_STATE_DICT
  fsdp_transformer_layer_cls_to_wrap: LlamaDecoderLayer
  fsdp_cpu_ram_efficient_loading: true
  fsdp_use_orig_params: true
mixed_precision: bf16
```

## Sharding Strategies
| Strategy | Memory Savings | Communication | When to Use |
|----------|---------------|---------------|-------------|
| FULL_SHARD | Maximum | Highest | Model doesn't fit on single GPU |
| SHARD_GRAD_OP | Moderate | Medium | Fits in memory but need gradient sharding |
| NO_SHARD | None (DDP) | Lowest | Model fits, maximize speed |

## Activation Checkpointing
```python
from torch.distributed.fsdp.wrap import transformer_auto_wrap_policy
from functools import partial

auto_wrap_policy = partial(
    transformer_auto_wrap_policy,
    transformer_layer_cls={LlamaDecoderLayer},
)
```

## Best Practices
- Use `SHARDED_STATE_DICT` for efficient checkpointing (avoids gathering full model)
- Enable `cpu_ram_efficient_loading` to avoid OOM during model loading
- Set `use_orig_params=True` when using optimizers that need parameter names
- Monitor per-GPU memory to verify sharding is working as expected
- Use `BACKWARD_PRE` prefetch for overlapping communication with compute
