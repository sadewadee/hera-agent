---
name: pytorch_custom_dataset
description: "Build custom PyTorch datasets and dataloaders"
version: "1.0"
trigger: "pytorch dataset dataloader custom data"
platforms: []
requires_tools: ["run_command"]
---

# Custom PyTorch Datasets

## Purpose
Build efficient custom datasets and data pipelines in PyTorch for training and evaluation.

## Instructions
1. Implement `__init__`, `__len__`, and `__getitem__` for map-style datasets
2. Or implement `__iter__` for iterable-style datasets (streaming)
3. Apply transforms and augmentations in `__getitem__`
4. Configure DataLoader with parallel workers and prefetching
5. Handle edge cases (missing files, corrupt data, variable-length sequences)

## Map-Style Dataset
```python
from torch.utils.data import Dataset

class TextClassificationDataset(Dataset):
    def __init__(self, texts, labels, tokenizer, max_length=512):
        self.texts = texts
        self.labels = labels
        self.tokenizer = tokenizer
        self.max_length = max_length

    def __len__(self):
        return len(self.texts)

    def __getitem__(self, idx):
        encoding = self.tokenizer(
            self.texts[idx],
            truncation=True,
            max_length=self.max_length,
            padding="max_length",
            return_tensors="pt",
        )
        return {
            "input_ids": encoding["input_ids"].squeeze(),
            "attention_mask": encoding["attention_mask"].squeeze(),
            "labels": torch.tensor(self.labels[idx], dtype=torch.long),
        }
```

## Collate Functions
```python
def dynamic_padding_collate(batch):
    max_len = max(len(item["input_ids"]) for item in batch)
    padded_batch = {
        "input_ids": torch.stack([
            F.pad(item["input_ids"], (0, max_len - len(item["input_ids"])))
            for item in batch
        ]),
        "labels": torch.stack([item["labels"] for item in batch]),
    }
    return padded_batch
```

## Iterable Dataset (Streaming)
```python
class StreamingDataset(IterableDataset):
    def __init__(self, file_paths):
        self.file_paths = file_paths

    def __iter__(self):
        worker_info = torch.utils.data.get_worker_info()
        if worker_info is not None:
            per_worker = len(self.file_paths) // worker_info.num_workers
            start = worker_info.id * per_worker
            paths = self.file_paths[start:start + per_worker]
        else:
            paths = self.file_paths
        for path in paths:
            yield from self._read_file(path)
```

## Best Practices
- Use `persistent_workers=True` and `pin_memory=True` in DataLoader
- Dynamic padding saves memory vs max-length padding
- Split file reading across workers for iterable datasets
- Cache preprocessed data to disk for expensive tokenization
- Use `torch.utils.data.Subset` for quick debugging with small data samples
