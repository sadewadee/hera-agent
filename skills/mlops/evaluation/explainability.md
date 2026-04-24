---
name: explainability
description: "ML model interpretability and explainability techniques"
version: "1.0"
trigger: "explainability interpretability shap lime"
platforms: []
requires_tools: ["run_command"]
---

# Model Explainability

## Purpose
Apply interpretability techniques to understand ML model predictions, debug model behavior, and satisfy regulatory requirements for explainable AI.

## Instructions
1. Determine the explanation scope (global vs local, model-agnostic vs model-specific)
2. Select appropriate explainability methods for the model type
3. Generate explanations for representative samples and edge cases
4. Validate explanations against domain knowledge
5. Present explanations in stakeholder-appropriate format

## Global Explainability
- **Feature importance**: Permutation importance, mean SHAP values
- **Partial Dependence Plots (PDP)**: Show marginal effect of a feature on prediction
- **Accumulated Local Effects (ALE)**: PDP alternative that handles correlated features
- **Global surrogate models**: Train interpretable model to approximate the black box

## Local Explainability
- **SHAP (SHapley Additive exPlanations)**: Game-theoretic attribution of feature contributions
- **LIME (Local Interpretable Model-agnostic Explanations)**: Local linear approximation
- **Counterfactual explanations**: Minimal changes to flip the prediction
- **Anchor explanations**: Sufficient conditions for a prediction

## Implementation
```python
import shap

explainer = shap.TreeExplainer(model)
shap_values = explainer.shap_values(X_test)

# Summary plot (global)
shap.summary_plot(shap_values, X_test)

# Force plot (local - single prediction)
shap.force_plot(explainer.expected_value, shap_values[0], X_test.iloc[0])
```

## Use Cases
- Regulatory compliance (GDPR right to explanation, Fair Lending)
- Model debugging (identify spurious correlations)
- Stakeholder trust building
- Feature engineering insights

## Best Practices
- Always validate explanations against domain expertise
- Use multiple methods and check for consistency
- Be cautious with high-dimensional or correlated features
- Document explanation limitations alongside the explanations
- Test explanations on adversarial inputs to verify robustness
