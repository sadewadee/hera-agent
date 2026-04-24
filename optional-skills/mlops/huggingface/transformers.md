---
name: huggingface_transformers
description: "HuggingFace Transformers library for model training and inference"
version: "1.0"
trigger: "huggingface transformers pretrained models"
platforms: []
requires_tools: ["run_command"]
---

# HuggingFace Transformers

## Purpose
Use HuggingFace Transformers for loading, fine-tuning, and deploying pre-trained models across NLP, vision, and multimodal tasks.

## Instructions
1. Select a pre-trained model from the HuggingFace Hub
2. Load model and tokenizer with Auto classes
3. Fine-tune with Trainer API or custom training loop
4. Evaluate on task-specific benchmarks
5. Push to Hub or export for deployment

## Quick Start
```python
from transformers import AutoTokenizer, AutoModelForSequenceClassification, Trainer, TrainingArguments

tokenizer = AutoTokenizer.from_pretrained("bert-base-uncased")
model = AutoModelForSequenceClassification.from_pretrained("bert-base-uncased", num_labels=2)

training_args = TrainingArguments(
    output_dir="./results",
    num_train_epochs=3,
    per_device_train_batch_size=16,
    per_device_eval_batch_size=64,
    evaluation_strategy="epoch",
    save_strategy="epoch",
    load_best_model_at_end=True,
    bf16=True,
)

trainer = Trainer(
    model=model,
    args=training_args,
    train_dataset=train_dataset,
    eval_dataset=eval_dataset,
    tokenizer=tokenizer,
)
trainer.train()
```

## Pipeline API (Quick Inference)
```python
from transformers import pipeline

classifier = pipeline("text-classification", model="distilbert-base-uncased-finetuned-sst-2-english")
result = classifier("This movie is great!")
```

## Model Types
- `AutoModelForCausalLM`: Text generation (GPT, LLaMA)
- `AutoModelForSeq2SeqLM`: Translation, summarization (T5, BART)
- `AutoModelForSequenceClassification`: Text classification
- `AutoModelForTokenClassification`: NER, POS tagging
- `AutoModelForQuestionAnswering`: Extractive QA
- `AutoModel`: Base model for custom heads

## Best Practices
- Use `device_map="auto"` for automatic model sharding across GPUs
- Use `torch_dtype=torch.bfloat16` to reduce memory usage
- Enable Flash Attention with `attn_implementation="flash_attention_2"`
- Use `gradient_checkpointing=True` to trade compute for memory
- Push models to Hub with `trainer.push_to_hub()`
