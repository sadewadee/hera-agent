---
name: model_card
description: "Create model cards for ML model documentation"
version: "1.0"
trigger: "model card documentation ml"
platforms: []
requires_tools: []
---

# Model Cards

## Purpose
Create comprehensive model cards that document ML model details, performance, limitations, and ethical considerations following industry best practices.

## Instructions
1. Gather model metadata (architecture, training data, hyperparameters)
2. Evaluate model across relevant metrics and demographic slices
3. Document intended use cases and out-of-scope applications
4. Identify potential biases and fairness concerns
5. Publish the model card alongside the model artifact

## Model Card Template

### Model Details
- Model name and version
- Model type (classification, regression, generative, etc.)
- Training framework and version
- Date trained and training duration
- License and citation information

### Intended Use
- Primary intended uses and users
- Out-of-scope use cases
- Deployment context (real-time, batch, edge)

### Training Data
- Dataset name, size, and source
- Data preprocessing steps
- Train/validation/test split ratios
- Known data quality issues or gaps

### Evaluation Results
- Metrics on held-out test set (accuracy, F1, AUC, RMSE, etc.)
- Performance broken down by demographic groups
- Performance on edge cases and adversarial inputs
- Comparison with baseline models

### Ethical Considerations
- Potential biases in training data
- Fairness across protected groups
- Privacy implications
- Environmental impact (training compute, carbon footprint)

### Limitations and Risks
- Known failure modes
- Performance degradation scenarios
- Data freshness requirements
- Scalability constraints

## Tools
- Hugging Face Model Card Metadata spec for standardized format
- Google Model Cards Toolkit for automated generation
- Evidently AI for bias and fairness reporting
