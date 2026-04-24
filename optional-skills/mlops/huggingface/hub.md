---
name: huggingface_hub
description: "HuggingFace Hub for model and dataset management"
version: "1.0"
trigger: "huggingface hub model sharing repository"
platforms: []
requires_tools: ["run_command"]
---

# HuggingFace Hub

## Purpose
Upload, download, version, and share ML models and datasets using HuggingFace Hub with Git-based version control.

## Instructions
1. Create a repository on HuggingFace Hub
2. Upload model files, tokenizer, and configuration
3. Write a model card with usage instructions
4. Manage access and visibility settings
5. Version models with branches and tags

## Upload Models
```python
from huggingface_hub import HfApi

api = HfApi()
api.create_repo("username/model-name", private=True)
api.upload_folder(
    folder_path="./model_output",
    repo_id="username/model-name",
    commit_message="Upload fine-tuned model v1",
)
```

## From Trainer
```python
trainer.push_to_hub("username/model-name", commit_message="Fine-tuned on custom dataset")
```

## Download Models
```python
from huggingface_hub import snapshot_download

snapshot_download(
    repo_id="username/model-name",
    local_dir="./models/my-model",
    revision="main",
)
```

## Model Cards
- Write README.md with YAML metadata header
- Include model description, training details, evaluation results
- Add inference widget configuration for web demo
- Tag with appropriate task, language, and license

## Access Control
- Public: Anyone can access
- Private: Only you and collaborators
- Gated: Users must accept terms before access
- Organization: Shared within a HuggingFace organization

## Best Practices
- Use `.gitignore` to exclude training artifacts
- Tag releases with semantic versioning (v1.0, v1.1)
- Include evaluation metrics in model card metadata
- Use Spaces for hosting interactive demos
- Set up webhooks for CI/CD on model updates
