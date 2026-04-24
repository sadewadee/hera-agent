---
name: data_loading
description: "Efficient data loading pipelines for ML training"
version: "1.0"
trigger: "data loading pipeline dataloader training"
platforms: []
requires_tools: ["run_command"]
---

# Efficient Data Loading

## Purpose
Build data loading pipelines that keep GPUs fully utilized by overlapping data preprocessing with model computation.

## Instructions
1. Profile data loading to identify bottlenecks (I/O vs CPU vs GPU)
2. Choose appropriate data format for your workload
3. Configure parallel data loading with prefetching
4. Optimize preprocessing with vectorized operations
5. Monitor GPU utilization to verify pipeline efficiency

## Data Formats
| Format | Best For | Random Access | Compression |
|--------|----------|---------------|-------------|
| WebDataset (.tar) | Large-scale streaming | No | Optional |
| HuggingFace Datasets | NLP, tabular | Yes (memory-mapped) | Yes |
| TFRecord | TensorFlow pipelines | No | Yes |
| Parquet | Tabular data | Yes (column-level) | Yes |
| LMDB | Image datasets | Yes | No |
| Mosaic StreamingDataset | Multi-node training | Yes | Yes |

## PyTorch DataLoader Optimization
```python
dataloader = DataLoader(
    dataset,
    batch_size=32,
    num_workers=4,           # parallel data loading processes
    pin_memory=True,         # faster CPU->GPU transfer
    prefetch_factor=2,       # batches prefetched per worker
    persistent_workers=True, # don't restart workers each epoch
    drop_last=True,          # consistent batch sizes
)
```

## num_workers Selection
- Start with `num_workers = 4 * num_gpus`
- Increase until CPU utilization saturates or no speed improvement
- Too many workers: context switching overhead, memory pressure
- Set `OMP_NUM_THREADS=1` to prevent OpenMP thread explosion

## Streaming Data
- Use WebDataset or Mosaic StreamingDataset for datasets larger than disk
- Stream from S3/GCS with multi-threaded prefetching
- Shuffle at the shard level, then within each shard
- Resume training from exact position using checkpoint index

## Profiling
- Use PyTorch Profiler to identify data loading bottlenecks
- Monitor GPU utilization (>90% indicates good data pipeline)
- If GPU utilization is low, data loading is the bottleneck
- Use `torch.utils.bottleneck` for quick profiling
