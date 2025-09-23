import glob
import json
import requests
from uuid import uuid4
from qdrant_client import QdrantClient
from qdrant_client.http.models import PointStruct

OLLAMA_EMBED_MODEL = "nomic-embed-text"
QDRANT_COLLECTION = "documents"

client = QdrantClient(url="http://localhost:6333")

def embed_text(text):
    response = requests.post(
        "http://localhost:11434/embed",
        json={"model": OLLAMA_EMBED_MODEL, "text": text}
    )
    response.raise_for_status()
    return response.json()["embedding"]

def ingest_docs():
    points = []
    for path in glob.glob("docs/*.txt"):
        with open(path, "r", encoding="utf-8") as f:
            text = f.read()

        # Simple chunking (split by 500 chars)
        for i in range(0, len(text), 500):
            chunk = text[i:i+500]
            vec = embed_text(chunk)
            points.append(PointStruct(id=str(uuid4()), vector=vec, payload={"text": chunk}))

    # Create collection if not exists
    client.create_collection(
        collection_name=QDRANT_COLLECTION,
        vector_size=len(points[0].vector),
        distance="Cosine"
    )

    # Upsert points
    client.upsert(collection_name=QDRANT_COLLECTION, points=points)
    print(f"Ingested {len(points)} document chunks into Qdrant.")

if __name__ == "__main__":
    ingest_docs()
