---
name: distillation
description: "Knowledge distillation from large to small models"
version: "1.0"
trigger: "distillation knowledge transfer teacher student"
platforms: []
requires_tools: ["run_command"]
---

# Knowledge Distillation

## Purpose
Transfer knowledge from a large teacher model to a smaller student model, achieving most of the teacher's performance at a fraction of the compute cost.

## Instructions
1. Train or obtain the teacher model (large, accurate)
2. Design the student model architecture (smaller, faster)
3. Generate soft labels from teacher predictions
4. Train student on combination of hard labels and teacher soft labels
5. Evaluate student against teacher and baseline models

## Distillation Loss
```python
# Combined loss: task loss + distillation loss
loss = alpha * task_loss(student_logits, hard_labels) + \
       (1 - alpha) * distill_loss(
           F.log_softmax(student_logits / temperature, dim=-1),
           F.softmax(teacher_logits / temperature, dim=-1)
       ) * (temperature ** 2)
```

## Key Parameters
- **Temperature (T)**: Higher values (2-20) soften probability distributions, revealing dark knowledge
- **Alpha**: Balance between task loss and distillation loss (0.3-0.7 typical)
- **Student architecture**: 2-10x smaller than teacher, same general structure family

## Distillation Strategies
- **Logit distillation**: Match output distributions (standard approach)
- **Feature distillation**: Match intermediate layer representations
- **Attention transfer**: Match attention maps between teacher and student
- **Self-distillation**: Teacher and student share architecture (different sizes)
- **Multi-teacher**: Ensemble of teachers for more robust knowledge

## LLM Distillation
- Use teacher to generate training data (synthetic data distillation)
- Chain-of-thought distillation: teach reasoning steps, not just answers
- Task-specific distillation: fine-tune student on teacher's task outputs
- Examples: Alpaca (GPT-4 -> LLaMA), Orca (GPT-4 -> 13B)

## Evaluation
- Compare student vs teacher on held-out test set
- Measure latency improvement and model size reduction
- Verify student generalizes (not just memorizing teacher outputs)
- Test on distribution shift scenarios
