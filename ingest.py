#!/usr/bin/env python3
import os
import glob
import requests
from uuid import uuid4

QDRANT_URL = os.environ.get("QDRANT_URL", "http://localhost:6333")
COLLECTION_NAME = os.environ.get("QDRANT_COLLECTION", "documents")
VECTOR_NAME = "embedding"   # named vector key
VECTOR_DIM = int(os.environ.get("VECTOR_DIM", "768"))


def create_collection():
    url = f"{QDRANT_URL}/collections/{COLLECTION_NAME}"
    body = {
        "vectors": {
            VECTOR_NAME: {
                "size": VECTOR_DIM,
                "distance": "Cosine"
            }
        }
    }
    resp = requests.put(url, json=body)
    print("Create collection:", resp.status_code, resp.text)


def upsert_points(points):
    url = f"{QDRANT_URL}/collections/{COLLECTION_NAME}/points?wait=true"
    body = {"points": points}
    resp = requests.put(url, json=body)
    resp.raise_for_status()
    return resp.json()


def embed_text(text: str):
    """
    Calls Ollama's embedding API (default: localhost:11434).
    Requires that you have `ollama` running and a model that supports embeddings
    e.g. `ollama pull nomic-embed-text` and then call it here.
    """
    url = "http://localhost:11434/api/embeddings"
    payload = {"model": "nomic-embed-text", "prompt": text}
    resp = requests.post(url, json=payload)
    resp.raise_for_status()
    data = resp.json()
    return data["embedding"]  # list of floats


def ingest_docs():
    create_collection()
    points = []
    for path in glob.glob("docs/*.txt"):
        with open(path, "r", encoding="utf-8") as f:
            text = f.read()
        # You can implement chunking if you like
        chunks = [text]
        for chunk in chunks:
            vec = embed_text(chunk)
            if len(vec) != VECTOR_DIM:
                raise ValueError(
                    f"Embedding dim {len(vec)} != configured {VECTOR_DIM}"
                )
            points.append({
                "id": str(uuid4()),
                "vector": vec,   
                "payload": {"text": chunk}
            })
            print(points)
    resp = upsert_points(points)
    print("Upsert result:", resp)


if __name__ == "__main__":
    ingest_docs()
