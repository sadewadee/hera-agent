---
name: faiss_gpu
description: "GPU-accelerated vector search with FAISS"
version: "1.0"
trigger: "faiss gpu vector search acceleration"
platforms: []
requires_tools: ["run_command"]
---

# FAISS GPU

## Purpose
Accelerate vector similarity search using FAISS GPU indexes for real-time applications requiring sub-millisecond latency.

## Instructions
1. Install faiss-gpu and verify CUDA availability
2. Create CPU index and transfer to GPU
3. Configure GPU resources and memory
4. Benchmark latency and throughput vs CPU
5. Handle multi-GPU sharding for large indexes

## Basic GPU Usage
```python
import faiss

# Create CPU index
d = 768
index_cpu = faiss.IndexFlatL2(d)

# Transfer to GPU
res = faiss.StandardGpuResources()
index_gpu = faiss.index_cpu_to_gpu(res, 0, index_cpu)  # GPU 0

# Use normally
index_gpu.add(vectors)
D, I = index_gpu.search(queries, k=10)
```

## Multi-GPU
```python
co = faiss.GpuMultipleClonerOptions()
co.shard = True  # shard index across GPUs
index_gpu = faiss.index_cpu_to_all_gpus(index_cpu, co=co)
```

## GPU Resource Configuration
```python
res = faiss.StandardGpuResources()
res.setTempMemory(1024 * 1024 * 1024)  # 1GB temp memory
res.setPinnedMemory(256 * 1024 * 1024)  # 256MB pinned memory
```

## Performance Comparison
| Index Type | CPU (ms) | GPU (ms) | Speedup |
|-----------|----------|----------|---------|
| Flat (1M vectors) | 50 | 2 | 25x |
| IVF4096,PQ32 (10M) | 5 | 0.5 | 10x |
| IVF16384,PQ64 (100M) | 10 | 1 | 10x |

## Best Practices
- GPU memory limits index size (A100 80GB holds ~100M 768-dim vectors)
- Use IVF+PQ for indexes larger than GPU memory
- Batch queries for maximum throughput (process 100+ queries at once)
- Use multi-GPU sharding for indexes exceeding single GPU memory
- Fall back to CPU for cold/infrequent queries to save GPU for training
