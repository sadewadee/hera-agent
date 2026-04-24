---
name: huggingface_datasets
description: "HuggingFace Datasets library for data loading and processing"
version: "1.0"
trigger: "huggingface datasets data loading processing"
platforms: []
requires_tools: ["run_command"]
---

# HuggingFace Datasets

## Purpose
Efficiently load, process, and share datasets using HuggingFace Datasets library with memory-mapped Apache Arrow backend.

## Instructions
1. Load datasets from Hub or local files
2. Apply preprocessing with `.map()` and `.filter()`
3. Configure streaming for large datasets
4. Handle train/validation/test splits
5. Push processed datasets to Hub for sharing

## Loading Data
```python
from datasets import load_dataset

# From Hub
dataset = load_dataset("imdb")

# From local files
dataset = load_dataset("csv", data_files="data.csv")
dataset = load_dataset("json", data_files="data.jsonl")
dataset = load_dataset("parquet", data_files="data.parquet")

# Streaming (no download)
dataset = load_dataset("c4", split="train", streaming=True)
```

## Preprocessing
```python
def tokenize_function(examples):
    return tokenizer(examples["text"], truncation=True, max_length=512)

tokenized = dataset.map(tokenize_function, batched=True, num_proc=4)
filtered = dataset.filter(lambda x: len(x["text"]) > 100)
```

## Memory Efficiency
- Arrow format: memory-mapped, zero-copy reads
- Streaming: process without downloading entire dataset
- `num_proc` for multiprocessing map operations
- `.with_format("torch")` for zero-copy PyTorch tensor conversion
- Cache is automatic; use `load_from_cache_file=False` to recompute

## Creating Datasets
```python
from datasets import Dataset, DatasetDict

# From dictionary
dataset = Dataset.from_dict({"text": texts, "label": labels})

# From pandas
dataset = Dataset.from_pandas(df)

# Train/test split
split = dataset.train_test_split(test_size=0.2, seed=42)
```

## Best Practices
- Use `batched=True` in `.map()` for 10-100x faster processing
- Set `num_proc` to CPU count for parallel preprocessing
- Use streaming for datasets larger than available RAM
- Cache tokenized datasets to avoid reprocessing
- Use `.select()` for quick debugging with small subsets
