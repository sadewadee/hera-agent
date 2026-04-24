---
name: drift_detection
description: "Detect data drift and model performance degradation"
version: "1.0"
trigger: "drift detection model monitoring data drift"
platforms: []
requires_tools: ["run_command"]
---

# Drift Detection

## Purpose
Monitor production ML models for data drift, concept drift, and performance degradation to trigger retraining or alerting.

## Instructions
1. Establish baseline distributions from training data
2. Configure drift detection monitors for input features and predictions
3. Set up alerting thresholds based on statistical tests
4. Automate retraining pipelines when drift exceeds thresholds
5. Track drift trends over time for capacity planning

## Types of Drift
- **Data drift (covariate shift)**: Input feature distributions change
- **Concept drift**: Relationship between features and target changes
- **Prediction drift**: Model output distribution changes
- **Label drift**: Ground truth label distribution changes

## Detection Methods
- **KL Divergence**: Measures difference between two probability distributions
- **KS Test (Kolmogorov-Smirnov)**: Non-parametric test for distribution equality
- **PSI (Population Stability Index)**: Quantifies distribution shift magnitude
- **ADWIN**: Adaptive windowing for streaming data drift detection
- **Page-Hinkley**: Sequential change-point detection

## Implementation Pattern
```python
from evidently import ColumnDriftMetric
from evidently.report import Report

report = Report(metrics=[
    ColumnDriftMetric(column_name="feature_1"),
    ColumnDriftMetric(column_name="feature_2"),
])
report.run(reference_data=train_df, current_data=production_df)
```

## Alerting Strategy
- **Warning**: PSI > 0.1 or KS p-value < 0.05 for any feature
- **Critical**: PSI > 0.25 or multiple features drifting simultaneously
- **Emergency**: Prediction distribution significantly shifted from baseline

## Response Playbook
1. Verify drift is real (not a data pipeline issue)
2. Identify which features are drifting and correlate with external events
3. If concept drift: retrain on recent data
4. If data drift: investigate upstream data source changes
5. Document and update monitoring thresholds
