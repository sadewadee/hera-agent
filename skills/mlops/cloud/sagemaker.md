---
name: sagemaker
description: "Deploy and manage ML models on AWS SageMaker"
version: "1.0"
trigger: "sagemaker aws ml deploy"
platforms: []
requires_tools: ["run_command"]
---

# AWS SageMaker

## Purpose
Deploy, train, and manage machine learning models using AWS SageMaker, including endpoint management, training jobs, and model registry.

## Instructions
1. Identify the ML workflow stage (training, deployment, monitoring)
2. Select the appropriate SageMaker service component
3. Configure the infrastructure and scaling parameters
4. Implement the workflow with proper IAM permissions and VPC settings
5. Set up monitoring and cost controls

## Training Jobs
- Use `sagemaker.estimator.Estimator` for custom training scripts
- Leverage built-in algorithms (XGBoost, BlazingText, etc.) when applicable
- Configure spot instances with `use_spot_instances=True` for cost savings up to 90%
- Set `max_run` and `max_wait` timeouts to prevent runaway costs
- Use SageMaker Debugger for profiling and detecting training issues

## Model Deployment
- Real-time endpoints for low-latency inference (<100ms)
- Serverless inference for intermittent traffic (cost-effective)
- Batch transform for large-scale offline predictions
- Multi-model endpoints to host multiple models on a single instance
- Use auto-scaling policies based on InvocationsPerInstance

## Model Registry
- Register model versions with approval workflows
- Tag models with metadata (accuracy, dataset version, training params)
- Use model groups to organize related model versions
- Automate promotion from staging to production with CI/CD

## Cost Optimization
- Use spot instances for training (set checkpointing for fault tolerance)
- Right-size endpoint instances based on load testing
- Delete unused endpoints (they bill per hour even with zero traffic)
- Use SageMaker Savings Plans for predictable workloads
- Monitor with CloudWatch and set billing alarms

## Common Patterns
```python
# Deploy a model to a real-time endpoint
from sagemaker.model import Model

model = Model(
    model_data="s3://bucket/model.tar.gz",
    role=role,
    image_uri=image_uri,
)
predictor = model.deploy(
    initial_instance_count=1,
    instance_type="ml.m5.xlarge",
    endpoint_name="my-endpoint",
)
```

## Troubleshooting
- Check CloudWatch logs at `/aws/sagemaker/Endpoints/<endpoint-name>`
- Verify IAM role has S3 read access for model artifacts
- Ensure VPC subnets have NAT gateway for internet access
- Use `describe_endpoint` to check endpoint status and failure reasons
