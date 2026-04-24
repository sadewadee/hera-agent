---
name: vertex_ai
description: "Build and deploy ML pipelines on Google Cloud Vertex AI"
version: "1.0"
trigger: "vertex ai gcp ml pipeline"
platforms: []
requires_tools: ["run_command"]
---

# Google Cloud Vertex AI

## Purpose
Build, deploy, and manage ML models and pipelines using Google Cloud Vertex AI, including AutoML, custom training, and model monitoring.

## Instructions
1. Determine the ML task type (classification, regression, NLP, vision, etc.)
2. Choose between AutoML and custom training based on requirements
3. Set up the Vertex AI pipeline with appropriate components
4. Deploy models with traffic splitting for A/B testing
5. Configure prediction monitoring and drift detection

## Custom Training
- Use `CustomTrainingJob` for single-node or `CustomContainerTrainingJob` for Docker
- Pre-built containers available for TensorFlow, PyTorch, XGBoost, Scikit-learn
- Configure machine type, accelerator (GPU/TPU), and replica count
- Use Vertex AI TensorBoard for experiment tracking
- Store artifacts in GCS with versioned paths

## Model Deployment
- Deploy to endpoints with traffic splitting between model versions
- Use private endpoints for VPC-internal predictions
- Configure autoscaling with min/max replica counts
- Batch prediction for large datasets via BigQuery or GCS
- Use model monitoring for feature skew and prediction drift

## Pipelines (Kubeflow)
- Define pipelines as Python functions with `@component` decorators
- Use `google_cloud_pipeline_components` for pre-built GCP steps
- Schedule pipelines with Cloud Scheduler
- Track lineage: dataset -> training -> model -> endpoint
- Cache pipeline steps to avoid redundant computation

## Feature Store
- Create feature groups from BigQuery tables or GCS files
- Serve features online with low-latency lookups
- Point-in-time correctness for training data
- Feature monitoring for distribution changes

## Cost Management
- Use preemptible/spot VMs for training jobs
- Set budget alerts on the GCP project
- Auto-scale endpoints to zero during idle periods
- Use smaller machine types for development, scale up for production
