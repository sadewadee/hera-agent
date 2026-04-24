---
name: triton
description: "NVIDIA Triton Inference Server deployment and management"
version: "1.0"
trigger: "triton inference server nvidia"
platforms: []
requires_tools: ["run_command"]
---

# NVIDIA Triton Inference Server

## Purpose
Deploy and manage ML models at scale using NVIDIA Triton Inference Server with support for multiple frameworks, dynamic batching, and model ensembles.

## Instructions
1. Prepare the model repository with correct directory structure
2. Write the model configuration file (config.pbtxt)
3. Launch Triton with appropriate GPU and memory settings
4. Configure dynamic batching and instance groups
5. Set up health checks, metrics, and load balancing

## Model Repository Structure
```
model_repository/
  model_name/
    config.pbtxt
    1/              # version 1
      model.onnx    # or model.plan, model.savedmodel, etc.
    2/              # version 2
      model.onnx
```

## Configuration
```protobuf
name: "my_model"
platform: "onnxruntime_onnx"
max_batch_size: 32
input [
  { name: "input_ids", data_type: TYPE_INT64, dims: [-1] }
]
output [
  { name: "logits", data_type: TYPE_FP32, dims: [-1, 50257] }
]
dynamic_batching {
  preferred_batch_size: [8, 16, 32]
  max_queue_delay_microseconds: 5000
}
instance_group [
  { count: 2, kind: KIND_GPU, gpus: [0] }
]
```

## Supported Backends
- TensorRT (highest performance for NVIDIA GPUs)
- ONNX Runtime (cross-platform, good performance)
- TensorFlow SavedModel
- PyTorch TorchScript
- Python backend (custom preprocessing/postprocessing)
- vLLM backend (for LLM serving)

## Model Ensembles
- Chain multiple models in a pipeline (preprocessing -> inference -> postprocessing)
- Define ensemble scheduler in config.pbtxt
- Each step can run on different hardware (CPU preprocessing, GPU inference)

## Monitoring
- Prometheus metrics endpoint at :8002/metrics
- Request count, latency, queue time, batch size histograms
- GPU utilization and memory via DCGM integration
- Health check endpoints for Kubernetes liveness/readiness probes
