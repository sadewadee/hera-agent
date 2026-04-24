---
name: chromadb_setup
description: "Set up and use ChromaDB for vector storage"
version: "1.0"
trigger: "chromadb chroma vector database embeddings"
platforms: []
requires_tools: ["run_command"]
---

# ChromaDB Setup

## Purpose
Set up and use ChromaDB as an embedded or client-server vector database for RAG applications and semantic search.

## Instructions
1. Install ChromaDB and choose deployment mode (embedded or client-server)
2. Create collections with appropriate distance functions
3. Add documents with embeddings and metadata
4. Query with vector similarity and metadata filtering
5. Persist data and manage collections

## Embedded Mode
```python
import chromadb

client = chromadb.PersistentClient(path="./chroma_data")
collection = client.get_or_create_collection(
    name="documents",
    metadata={"hnsw:space": "cosine"},
)
```

## Adding Documents
```python
collection.add(
    documents=["doc1 text", "doc2 text"],
    metadatas=[{"source": "web"}, {"source": "pdf"}],
    ids=["doc1", "doc2"],
)
# ChromaDB auto-generates embeddings using default model
```

## Custom Embeddings
```python
from chromadb.utils import embedding_functions

ef = embedding_functions.SentenceTransformerEmbeddingFunction(
    model_name="all-MiniLM-L6-v2"
)
collection = client.get_or_create_collection(
    name="documents",
    embedding_function=ef,
)
```

## Querying
```python
results = collection.query(
    query_texts=["search query"],
    n_results=10,
    where={"source": "web"},
    include=["documents", "metadatas", "distances"],
)
```

## Client-Server Mode
```bash
# Start server
chroma run --host 0.0.0.0 --port 8000 --path ./chroma_data

# Connect from client
client = chromadb.HttpClient(host="localhost", port=8000)
```

## Best Practices
- Use PersistentClient for data that must survive restarts
- Choose cosine distance for normalized embeddings
- Use metadata filters to narrow search before vector similarity
- Batch inserts for performance (1000+ documents at a time)
- Monitor collection size and plan for scaling
