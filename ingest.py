#!/usr/bin/env python3
"""
ingest.py
- Reads text files from ./docs/*.txt
- Obtains embeddings via Ollama (localhost:11434)
- Creates Qdrant collection `documents` with the correct vector size
- Upserts points using unnamed vectors (vector: [ ... ])
"""
import os
import glob
import json
import requests
from uuid import uuid4

QDRANT_URL = os.environ.get("QDRANT_URL", "http://localhost:6333")
COLLECTION_NAME = os.environ.get("QDRANT_COLLECTION", "documents")
OLLAMA_URL = os.environ.get("OLLAMA_URL", "http://localhost:11434/api/embeddings")
EMBED_MODEL = os.environ.get("EMBED_MODEL", "nomic-embed-text")
# set to None to auto-detect from embed result:
VECTOR_DIM = os.environ.get("VECTOR_DIM", "")

def embed_text(text: str):
    """Call Ollama embedding endpoint and return list[float]."""
    payload = {"model": EMBED_MODEL, "prompt": text}
    resp = requests.post(OLLAMA_URL, json=payload, timeout=30)
    resp.raise_for_status()
    data = resp.json()
    if "embedding" in data:
        return data["embedding"]
    # some models return {"embeddings": [...]} or other shapes; try common alternatives:
    if "embeddings" in data and isinstance(data["embeddings"], list) and len(data["embeddings"])>0:
        return data["embeddings"][0]
    raise RuntimeError(f"Unexpected embedding response shape: {data}")

def create_collection(vector_size: int):
    """
    Create a collection with unnamed dense vector of given size.
    This creates the collection at: PUT /collections/<COLLECTION_NAME>
    """
    url = f"{QDRANT_URL}/collections/{COLLECTION_NAME}"
    body = {
        "vectors": {
            # Unnamed vectors mode: provide "size" directly.
            "size": vector_size,
            "distance": "Cosine"
        }
    }
    # Qdrant 1.x accepts body like {"vectors": {"size": 768, ...}} for unnamed mode.
    resp = requests.put(url, json=body, timeout=10)
    if resp.status_code in (200, 201):
        print(f"[ok] collection '{COLLECTION_NAME}' created/updated (size={vector_size})")
    else:
        # If exists, Qdrant may return 400/409 - still print full response for debugging
        print(f"[collection create] status={resp.status_code} body={resp.text}")
        resp.raise_for_status()

def upsert_points(points):
    """
    Upsert points with unnamed vector format:
      { "points": [ { "id": "...", "vector": [...], "payload": {...} }, ... ] }
    """
    url = f"{QDRANT_URL}/collections/{COLLECTION_NAME}/points?wait=true"
    body = {"points": points}
    resp = requests.put(url, json=body, timeout=30)
    if not resp.ok:
        print("[upsert] qdrant response:", resp.status_code, resp.text)
    resp.raise_for_status()
    return resp.json()

def ingest_docs():
    files = sorted(glob.glob("docs/*.txt"))
    if not files:
        print("No docs found in ./docs. Place .txt files there.")
        return

    # sample embedding to detect dimension (unless VECTOR_DIM env var set)
    sample_text = ""
    with open(files[0], "r", encoding="utf-8") as f:
        sample_text = f.read()[:2000] or "sample"
    sample_vec = embed_text(sample_text)
    detected_dim = len(sample_vec)
    env_dim = int(VECTOR_DIM) if VECTOR_DIM else detected_dim
    if env_dim != detected_dim:
        print(f"VECTOR_DIM env {VECTOR_DIM} does not match detected dim {detected_dim}. Using detected {detected_dim}.")
        env_dim = detected_dim

    # create collection if needed
    create_collection(env_dim)

    # build points
    points = []
    for path in files:
        with open(path, "r", encoding="utf-8") as f:
            text = f.read()
        # basic chunking: for now use whole file as one chunk. Replace with real chunker if desired.
        chunks = [text]
        for chunk in chunks:
            vec = embed_text(chunk)
            if len(vec) != env_dim:
                raise ValueError(f"Embedding dim {len(vec)} != configured collection dim {env_dim}")
            points.append({
                "id": str(uuid4()),
                "vector": vec,           # UNNAMED vector format
                "payload": {"text": chunk, "source": os.path.basename(path)}
            })
    print(f"Upserting {len(points)} points to collection '{COLLECTION_NAME}' ...")
    resp = upsert_points(points)
    print("Upsert response:", resp)

if __name__ == "__main__":
    ingest_docs()
