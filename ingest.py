import requests
import glob
from uuid import uuid4

QDRANT_URL = "http://localhost:6333"
COLLECTION = "docs"

def embed_text(text: str):
    url = "http://localhost:11434/api/embeddings"
    body = {
        "model": "nomic-embed-text",  
        "input": text
    }
    r = requests.post(url, json=body)
    r.raise_for_status()
    return r.json()["embedding"]

def chunk_text(text, max_length=500):
    # simple splitter
    words = text.split()
    for i in range(0, len(words), max_length):
        yield " ".join(words[i:i+max_length])

def create_collection():
    url = f"{QDRANT_URL}/collections/{COLLECTION}"
    body = {
        "vectors": {"size": 768, "distance": "Cosine"}
    }
    r = requests.put(url, json=body)
    print("Collection create:", r.status_code, r.text)

def upsert_points(points):
    url = f"{QDRANT_URL}/collections/{COLLECTION}/points?wait=true"
    body = {"points": points}
    r = requests.put(url, json=body)
    if r.status_code not in (200, 201):
        print("Upsert error:", r.status_code, r.text)
    return r

def ingest_docs():
    create_collection()
    points = []
    for path in glob.glob("docs/*.txt"):
        with open(path, "r", encoding="utf-8") as f:
            text = f.read()
        for chunk in chunk_text(text):
            vec = embed_text(chunk)
            points.append({
                "id": str(uuid4()),
                "vector": vec,
                "payload": {"text": chunk}
            })
    resp = upsert_points(points)
    print("Upsert response:", resp.text)

if __name__ == "__main__":
    ingest_docs()
