---
name: checkpointing
description: "Training checkpoint strategies and recovery"
version: "1.0"
trigger: "checkpointing training save resume recovery"
platforms: []
requires_tools: ["run_command"]
---

# Training Checkpointing

## Purpose
Save and resume training state to recover from failures, enable long-running jobs on preemptible hardware, and maintain training artifacts.

## Instructions
1. Configure checkpoint frequency based on job duration and failure risk
2. Save complete training state (model, optimizer, scheduler, epoch, RNG state)
3. Implement checkpoint rotation to manage storage
4. Test checkpoint resume before relying on it for long jobs
5. Store checkpoints in durable storage (S3, GCS) for cloud training

## What to Save
```python
checkpoint = {
    'epoch': epoch,
    'global_step': global_step,
    'model_state_dict': model.state_dict(),
    'optimizer_state_dict': optimizer.state_dict(),
    'scheduler_state_dict': scheduler.state_dict(),
    'loss': loss,
    'rng_state': torch.random.get_rng_state(),
    'cuda_rng_state': torch.cuda.get_rng_state_all(),
    'config': training_config,
}
torch.save(checkpoint, f'checkpoint_step_{global_step}.pt')
```

## Resume Pattern
```python
if resume_path:
    checkpoint = torch.load(resume_path)
    model.load_state_dict(checkpoint['model_state_dict'])
    optimizer.load_state_dict(checkpoint['optimizer_state_dict'])
    scheduler.load_state_dict(checkpoint['scheduler_state_dict'])
    start_epoch = checkpoint['epoch']
    torch.random.set_rng_state(checkpoint['rng_state'])
```

## Checkpoint Rotation
- Keep last N checkpoints (3-5 typical)
- Keep checkpoints at regular milestones (every 10% of training)
- Keep the best checkpoint (lowest validation loss)
- Delete intermediate checkpoints to manage storage

## Distributed Training Checkpoints
- Only save from rank 0 to avoid duplicate writes
- Use `torch.distributed.barrier()` before and after saving
- For FSDP: use `StateDictType.FULL_STATE_DICT` for portable checkpoints
- For DeepSpeed: use `model.save_checkpoint()` which handles sharding

## Best Practices
- Save every 15-30 minutes for preemptible instances
- Always verify resume works by running a short training job with save/load
- Include training config in checkpoint for reproducibility
- Use async checkpoint saving to avoid blocking training
- Compress checkpoints if storage is a concern (model weights compress 2-3x)
