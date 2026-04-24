---
name: faiss
description: "Facebook AI Similarity Search (FAISS) for vector indexing"
version: "1.0"
trigger: "faiss vector index similarity search"
platforms: []
requires_tools: ["run_command"]
---

# FAISS

## Purpose
Build high-performance vector similarity search indexes using Facebook's FAISS library for offline and real-time nearest neighbor retrieval.

## Instructions
1. Choose the appropriate index type based on dataset size and requirements
2. Train the index on representative vectors (for quantization-based indexes)
3. Add vectors to the index
4. Search with query vectors and retrieve top-k neighbors
5. Tune parameters for your accuracy/speed/memory trade-off

## Index Selection Guide
| Dataset Size | Index Type | Memory | Speed | Recall |
|-------------|------------|--------|-------|--------|
| <10K        | IndexFlatL2 | High  | Fast  | 100%   |
| 10K-1M      | IndexIVFFlat | Medium | Fast | 95%+  |
| 1M-100M     | IndexIVFPQ | Low    | Very fast | 90%+ |
| 100M+       | IndexIVFPQ + OPQ | Very low | Very fast | 85%+ |
| Any (GPU)   | GpuIndexFlatL2 | High | Fastest | 100% |

## Basic Usage
```python
import faiss
import numpy as np

d = 768  # dimension
index = faiss.IndexFlatL2(d)  # exact search
index.add(vectors)  # add vectors (numpy float32 array)

D, I = index.search(query_vectors, k=10)  # search top-10
# D = distances, I = indices
```

## IVF + PQ Index (Large Scale)
```python
nlist = 1024  # number of Voronoi cells
m = 32        # number of sub-quantizers
bits = 8      # bits per sub-quantizer

quantizer = faiss.IndexFlatL2(d)
index = faiss.IndexIVFPQ(quantizer, d, nlist, m, bits)
index.train(training_vectors)  # train on representative sample
index.add(vectors)
index.nprobe = 32  # search 32 cells (trade-off recall vs speed)
```

## GPU Acceleration
```python
res = faiss.StandardGpuResources()
gpu_index = faiss.index_cpu_to_gpu(res, 0, cpu_index)
```

## Best Practices
- Always normalize vectors before using inner product similarity
- Train IVF indexes on at least 30x nlist vectors
- Tune nprobe at query time: higher = better recall, slower search
- Use OPQ (Optimized Product Quantization) for better PQ quality
- Memory-map large indexes with `faiss.read_index` and `IO_FLAG_MMAP`
