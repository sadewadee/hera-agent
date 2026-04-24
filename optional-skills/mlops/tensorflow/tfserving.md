---
name: tfserving
description: "Deploy models with TensorFlow Serving"
version: "1.0"
trigger: "tensorflow serving deployment production"
platforms: []
requires_tools: ["run_command"]
---

# TensorFlow Serving

## Purpose
Deploy TensorFlow models in production using TensorFlow Serving with REST and gRPC endpoints, batching, and model versioning.

## Instructions
1. Export model as SavedModel format
2. Set up model directory with versioning structure
3. Launch TensorFlow Serving with Docker or binary
4. Configure batching and model warmup
5. Monitor with Prometheus metrics

## Export SavedModel
```python
model.save("/models/my_model/1/")  # version 1
# Directory structure:
# /models/my_model/
#   1/
#     saved_model.pb
#     variables/
```

## Docker Deployment
```bash
docker run -p 8501:8501 -p 8500:8500 \
  -v /models:/models \
  -e MODEL_NAME=my_model \
  tensorflow/serving:latest
```

## REST API
```bash
curl -X POST http://localhost:8501/v1/models/my_model:predict \
  -H "Content-Type: application/json" \
  -d '{"instances": [[1.0, 2.0, 3.0]]}'
```

## Batching Configuration
```
max_batch_size { value: 32 }
batch_timeout_micros { value: 5000 }
max_enqueued_batches { value: 100 }
num_batch_threads { value: 4 }
```

## Model Versioning
- TF Serving auto-detects new versions in the model directory
- Latest version is served by default
- Pin specific versions with `model_version_policy`
- Warmup new versions before serving with `tf_serving_warmup`

## Best Practices
- Use gRPC for internal services (lower latency than REST)
- Enable request batching for GPU-served models
- Use model warmup to avoid cold-start latency
- Monitor with /monitoring/prometheus/metrics endpoint
- Set --per_process_gpu_memory_fraction to share GPU with other services
