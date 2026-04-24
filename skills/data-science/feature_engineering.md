---
name: feature_engineering
description: "Feature engineering techniques"
version: "1.0"
trigger: "feature engineering transformation selection"
platforms: []
requires_tools: ["run_command"]
---

# Feature engineering techniques

## Purpose
Apply feature engineering techniques to extract insights from data, supporting both exploratory analysis and production data pipelines.

## Instructions
1. Understand the data and business question
2. Explore data quality, distributions, and relationships
3. Select and apply appropriate analytical methods
4. Validate results with proper statistical rigor
5. Communicate findings clearly with visualizations

## Key Methods
- Descriptive statistics and summary metrics
- Hypothesis testing and confidence intervals
- Correlation and causation analysis
- Cross-validation for model selection
- Feature importance and interpretability

## Workflow
1. Data ingestion and initial profiling
2. Exploratory data analysis (EDA)
3. Data cleaning and transformation
4. Feature engineering and selection
5. Model building or analysis execution
6. Evaluation and validation
7. Reporting and visualization

## Tools
- Python: pandas, numpy, scipy, scikit-learn, matplotlib, seaborn
- R: tidyverse, ggplot2, caret
- SQL: window functions, CTEs, analytical functions
- Visualization: plotly, altair, d3.js

## Best Practices
- Always start with EDA before modeling
- Check assumptions of statistical tests before applying
- Use cross-validation rather than single train/test splits
- Report confidence intervals, not just point estimates
- Document data lineage and transformation steps
- Version control data processing pipelines
