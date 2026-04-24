---
name: mlflow
description: "MLflow experiment tracking and model registry"
version: "1.0"
trigger: "mlflow experiment tracking model registry"
platforms: []
requires_tools: ["run_command"]
---

# MLflow experiment tracking and model registry

## Purpose
MLflow experiment tracking and model registry for production ML workflows and model lifecycle management.

## Instructions
1. Install and configure the platform
2. Set up project structure and configuration
3. Implement ML workflow integration
4. Configure model serving and monitoring
5. Deploy and manage in production

## Setup
- Install from package manager or container
- Configure storage backends and metadata stores
- Set up authentication and access control
- Integrate with existing ML frameworks
- Configure logging and monitoring

## Workflow Integration
- Experiment tracking and comparison
- Model versioning and registry
- Pipeline orchestration
- Artifact management and lineage
- Automated retraining triggers

## Deployment
- Model serving with REST/gRPC endpoints
- A/B testing and canary deployments
- Auto-scaling based on traffic
- Health checks and monitoring
- Rollback strategies

## Monitoring
- Model performance metrics
- Data drift detection
- Resource utilization
- Request latency and throughput
- Error rates and alerts

## Best Practices
- Version everything (code, data, models, config)
- Automate the ML pipeline end-to-end
- Monitor models in production continuously
- Test model behavior before deployment
- Document experiments and decisions
