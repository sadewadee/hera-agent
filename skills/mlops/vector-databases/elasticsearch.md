---
name: elasticsearch
description: "Elasticsearch for vector search and hybrid retrieval"
version: "1.0"
trigger: "elasticsearch vector search knn"
platforms: []
requires_tools: ["run_command"]
---

# Elasticsearch Vector Search

## Purpose
Use Elasticsearch for vector similarity search combined with traditional full-text search for hybrid retrieval in RAG and search applications.

## Instructions
1. Create an index with dense_vector field mapping
2. Configure HNSW parameters for your accuracy/speed trade-off
3. Index documents with both text and embedding vectors
4. Query using kNN search, text search, or hybrid combination
5. Monitor search latency and relevance metrics

## Index Mapping
```json
{
  "mappings": {
    "properties": {
      "title": { "type": "text" },
      "content": { "type": "text" },
      "embedding": {
        "type": "dense_vector",
        "dims": 768,
        "index": true,
        "similarity": "cosine"
      }
    }
  }
}
```

## kNN Search
```json
{
  "knn": {
    "field": "embedding",
    "query_vector": [0.1, 0.2, ...],
    "k": 10,
    "num_candidates": 100
  }
}
```

## Hybrid Search (Vector + Text)
```json
{
  "query": {
    "bool": {
      "should": [
        { "match": { "content": "machine learning transformers" } }
      ],
      "filter": [
        { "term": { "category": "research" } }
      ]
    }
  },
  "knn": {
    "field": "embedding",
    "query_vector": [0.1, 0.2, ...],
    "k": 10,
    "num_candidates": 100,
    "boost": 0.5
  }
}
```

## HNSW Tuning
- `m`: Number of connections per node (16 default, higher = better recall, more memory)
- `ef_construction`: Build-time beam width (100 default, higher = better index quality)
- `ef_search`: Query-time beam width (trade-off between latency and recall)

## Best Practices
- Use cosine similarity for normalized embeddings, dot_product for unnormalized
- Pre-filter with metadata before kNN to reduce candidate set
- Benchmark with your actual data - synthetic benchmarks mislead
- Monitor segment merges which can cause latency spikes
- Use quantized vectors (byte or binary) for memory reduction
