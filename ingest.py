#!/usr/bin/env python3
"""
ingest.py
- Walks docs/ for text files
- Chunks them
- Produces embeddings (via ollama CLI fallback)
- Upserts into Qdrant collection "local_rag"
"""

import os
import json
import math
import subprocess
import requests
from typing import List, Dict
from pathlib import Path
import uuid


from qdrant_client import QdrantClient
from qdrant_client.http.models import (
    VectorParams,
    Distance,
    PointStruct,
)

# CONFIG (make editable via env)
QDRANT_HOST = os.environ.get("QDRANT_HOST", "127.0.0.1")
QDRANT_PORT = int(os.environ.get("QDRANT_PORT", "6333"))
COLLECTION_NAME = os.environ.get("QDRANT_COLLECTION", "local_rag")
EMBED_DIM = int(os.environ.get("EMBED_DIM", "768"))  # update if your embed model uses different dim
CHUNK_SIZE = int(os.environ.get("CHUNK_SIZE", "800"))  # characters
CHUNK_OVERLAP = int(os.environ.get("CHUNK_OVERLAP", "200"))  # characters

client = QdrantClient(host=QDRANT_HOST, port=QDRANT_PORT)


def ensure_collection():
    collections = client.get_collections().collections
    if any(c.name == COLLECTION_NAME for c in collections):
        print(f"Collection {COLLECTION_NAME} exists.")
        return
    print(f"Creating collection {COLLECTION_NAME} with vector size {EMBED_DIM}")
    client.create_collection(
        collection_name=COLLECTION_NAME,
        vectors_config=VectorParams(size=EMBED_DIM, distance=Distance.COSINE),
    )


def chunk_text(text: str, chunk_size=CHUNK_SIZE, overlap=CHUNK_OVERLAP) -> List[str]:
    if len(text) <= chunk_size:
        return [text]
    chunks = []
    start = 0
    while start < len(text):
        end = min(start + chunk_size, len(text))
        chunk = text[start:end]
        chunks.append(chunk)
        if end == len(text):
            break
        start = end - overlap
    return chunks


def embed_texts(texts):
    """
    Generate embeddings using Ollama's HTTP API instead of CLI.
    Requires Ollama running locally (default http://localhost:11434).
    """
    vectors = []
    for t in texts:
        r = requests.post(
            "http://localhost:11434/api/embeddings",
            json={
                "model": "nomic-embed-text",
                "prompt": t
            },
            timeout=60
        )
        r.raise_for_status()
        data = r.json()
        vectors.append(data["embedding"])
    return vectors


def ingest_folder(folder: str = "docs"):
    ensure_collection()
    points: List[PointStruct] = []
    uid = 0
    folder_path = Path(folder)
    if not folder_path.exists():
        print("No docs/ folder found. Create docs/ and add text files.")
        return
    for p in folder_path.rglob("*"):
        if p.is_file() and p.suffix.lower() in (".txt", ".md"):
            text = p.read_text(encoding="utf-8")
            chunks = chunk_text(text)
            vectors = embed_texts(chunks)
            for i, chunk in enumerate(chunks):
                point_id = str(uuid.uuid4())
                payload = {"text": chunk, "source": str(p)}
                vec = vectors[i]
                points.append(PointStruct(id=point_id, vector=vec, payload=payload))
                uid += 1
            # batch upserts to qdrant each file
            if points:
                client.upsert(collection_name=COLLECTION_NAME, points=points)
                print(f"Upserted {len(points)} points from {p}")
                points.clear()
    print("Ingest finished.")


if __name__ == "__main__":
    ingest_folder("docs")
