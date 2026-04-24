---
name: evaluation_pipeline
description: "Automated model evaluation pipelines"
version: "1.0"
trigger: "model evaluation pipeline automated testing"
platforms: []
requires_tools: ["run_command"]
---

# Model Evaluation Pipelines

## Purpose
Build automated evaluation pipelines that test ML models across multiple benchmarks, datasets, and criteria before deployment.

## Instructions
1. Define evaluation criteria and benchmarks for your task
2. Build automated evaluation harness with reproducible metrics
3. Set pass/fail thresholds for each metric
4. Integrate evaluation into CI/CD pipeline
5. Archive evaluation results with model artifacts

## Evaluation Framework
```python
class EvaluationPipeline:
    def __init__(self, model, benchmarks, thresholds):
        self.model = model
        self.benchmarks = benchmarks
        self.thresholds = thresholds

    def run(self):
        results = {}
        for benchmark in self.benchmarks:
            results[benchmark.name] = benchmark.evaluate(self.model)
        return self.check_thresholds(results)
```

## Standard Benchmarks by Task
- **NLP**: GLUE, SuperGLUE, SQuAD, MMLU, HellaSwag
- **Vision**: ImageNet, COCO, ADE20K
- **Code**: HumanEval, MBPP, SWE-bench
- **Safety**: TruthfulQA, ToxiGen, BBQ (bias)
- **Multilingual**: XNLI, XQuAD, FLORES

## Evaluation Dimensions
- **Accuracy**: Task-specific metrics (F1, BLEU, accuracy)
- **Robustness**: Performance on adversarial/perturbed inputs
- **Fairness**: Performance parity across demographic groups
- **Latency**: Inference speed at target batch size
- **Cost**: Compute cost per prediction

## CI/CD Integration
- Run evaluation on every model training completion
- Gate deployment on threshold checks passing
- Compare new model against current production model
- Generate evaluation reports as pipeline artifacts
- Alert on regression beyond acceptable margins

## Best Practices
- Use held-out test sets never seen during training or hyperparameter tuning
- Include both automatic metrics and human evaluation for generative tasks
- Track metrics over time to detect gradual degradation
- Test on realistic production-like data, not just academic benchmarks
