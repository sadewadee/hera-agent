---
name: torchserve
description: "Deploy PyTorch models with TorchServe"
version: "1.0"
trigger: "torchserve deployment serving pytorch"
platforms: []
requires_tools: ["run_command"]
---

# TorchServe

## Purpose
Deploy PyTorch models as scalable REST endpoints using TorchServe with custom handlers, batching, and monitoring.

## Instructions
1. Archive the model with torch-model-archiver
2. Write a custom handler for preprocessing and postprocessing
3. Configure TorchServe settings (workers, batch size, timeout)
4. Start the server and test endpoints
5. Monitor with the management API and metrics endpoint

## Model Archiving
```bash
torch-model-archiver \
  --model-name my_model \
  --version 1.0 \
  --serialized-file model.pt \
  --handler handler.py \
  --extra-files config.json,vocab.txt \
  --export-path model_store/
```

## Custom Handler
```python
from ts.torch_handler.base_handler import BaseHandler
import torch

class MyHandler(BaseHandler):
    def preprocess(self, data):
        texts = [d.get("body").decode("utf-8") for d in data]
        inputs = self.tokenizer(texts, padding=True, return_tensors="pt")
        return inputs.to(self.device)

    def inference(self, inputs):
        with torch.no_grad():
            outputs = self.model(**inputs)
        return outputs.logits

    def postprocess(self, outputs):
        predictions = outputs.argmax(dim=-1).tolist()
        return [{"label": self.labels[p]} for p in predictions]
```

## Configuration (config.properties)
```properties
inference_address=http://0.0.0.0:8080
management_address=http://0.0.0.0:8081
metrics_address=http://0.0.0.0:8082
number_of_netty_threads=4
job_queue_size=100
model_store=/models
load_models=all
```

## Batch Inference
```properties
# In model config
batch_size=32
max_batch_delay=100  # ms to wait for batch formation
```

## Monitoring
- Metrics endpoint at :8082/metrics (Prometheus format)
- Management API at :8081 for model CRUD operations
- Health check at :8080/ping
- Log customization via log4j2.xml
