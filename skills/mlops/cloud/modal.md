---
name: modal
description: "Serverless ML infrastructure with Modal"
version: "1.0"
trigger: "modal serverless ml"
platforms: []
requires_tools: ["run_command"]
---

# Modal

## Purpose
Deploy and run ML workloads on Modal's serverless GPU infrastructure with minimal configuration and pay-per-second billing.

## Instructions
1. Define your Modal app with appropriate decorators
2. Specify container image and GPU requirements
3. Configure volumes for model weights and data
4. Deploy as a web endpoint, cron job, or on-demand function
5. Monitor execution and costs in the Modal dashboard

## Core Concepts
- **Stubs/Apps**: Entry point for defining Modal functions
- **Images**: Container images built from pip/conda/Dockerfile
- **Volumes**: Persistent storage for model weights and datasets
- **Secrets**: Secure environment variable injection
- **GPU selection**: T4, A10G, A100, H100 with fractional GPU support

## Deployment Patterns
```python
import modal

app = modal.App("my-ml-app")

image = modal.Image.debian_slim().pip_install("torch", "transformers")

@app.function(image=image, gpu="A100", timeout=600)
def inference(prompt: str) -> str:
    from transformers import pipeline
    pipe = pipeline("text-generation", model="meta-llama/Llama-2-7b-hf")
    return pipe(prompt, max_new_tokens=256)[0]["generated_text"]

@app.local_entrypoint()
def main():
    result = inference.remote("Hello, world!")
    print(result)
```

## Web Endpoints
- Use `@app.web_endpoint()` for HTTP endpoints
- Use `@app.asgi_app()` for FastAPI/Starlette apps
- Auto-scaling from zero to handle traffic spikes
- Custom domains via Modal dashboard

## Best Practices
- Use `modal.Volume` for caching model weights between invocations
- Pre-download models in the image build step for faster cold starts
- Use `modal.Secret` for API keys, never hardcode
- Set appropriate timeouts and memory limits
- Use `@app.cls()` for stateful services that load models once
