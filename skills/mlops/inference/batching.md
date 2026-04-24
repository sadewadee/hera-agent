---
name: batching
description: "Inference request batching strategies"
version: "1.0"
trigger: "batching inference throughput dynamic batch"
platforms: []
requires_tools: ["run_command"]
---

# Inference Batching

## Purpose
Maximize GPU utilization and throughput by batching inference requests, balancing latency requirements with resource efficiency.

## Instructions
1. Profile single-request latency and GPU utilization
2. Determine acceptable latency budget for the use case
3. Choose a batching strategy (static, dynamic, continuous)
4. Configure batch size, timeout, and queue depth parameters
5. Monitor p50/p95/p99 latency and throughput under load

## Batching Strategies
- **Static batching**: Fixed batch size, wait until batch is full or timeout
- **Dynamic batching**: Adaptive batch size based on queue depth and latency
- **Continuous batching (iteration-level)**: For autoregressive models, add new requests mid-generation
- **Sequence bucketing**: Group similar-length sequences to minimize padding waste

## Configuration Parameters
- `max_batch_size`: Upper limit on requests per batch (8-64 typical)
- `batch_timeout_ms`: Maximum wait time for batch formation (5-50ms typical)
- `max_queue_depth`: Reject requests when queue exceeds this depth
- `preferred_batch_size`: Target batch size for optimal GPU utilization

## Continuous Batching (LLMs)
- Essential for autoregressive generation where sequences finish at different times
- Supported by vLLM, TGI, TensorRT-LLM
- Increases throughput 2-10x compared to static batching
- Uses paged attention (vLLM) to manage KV cache memory efficiently

## Monitoring Metrics
- Batch formation time (time spent waiting for batch)
- Average batch size (utilization indicator)
- Queue depth over time (capacity planning)
- Latency per request (p50, p95, p99)
- Throughput (requests/second, tokens/second for LLMs)

## Trade-offs
- Larger batches: higher throughput, higher latency, more memory
- Smaller batches: lower latency, lower GPU utilization
- Timeout too low: small batches, wasted GPU cycles
- Timeout too high: latency spikes, poor user experience
