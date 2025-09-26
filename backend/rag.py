import os
import requests
import faiss
from sentence_transformers import SentenceTransformer
from typing import List

DEEPSEEK_API_KEY = os.getenv("DEEPSEEK_API_KEY")
DEEPSEEK_API_URL = "https://api.deepseek.com/v1/chat/completions"

model = SentenceTransformer("all-MiniLM-L6-v2")

# Load and embed documents
with open("data/documents.txt", "r", encoding="utf-8") as f:
    docs = f.readlines()
doc_embeddings = model.encode(docs)
index = faiss.IndexFlatL2(len(doc_embeddings[0]))
index.add(doc_embeddings)

def retrieve_context(query: str, k: int = 3) -> List[str]:
    query_embedding = model.encode([query])
    _, indices = index.search(query_embedding, k)
    return [docs[i] for i in indices[0]]

def get_rag_response(query: str) -> str:
    context = retrieve_context(query)
    prompt = f"Context:\n{''.join(context)}\n\nQuestion: {query}\nAnswer:"
    headers = {"Authorization": f"Bearer {DEEPSEEK_API_KEY}"}
    payload = {
        "model": "deepseek-chat",
        "messages": [{"role": "user", "content": prompt}],
        "temperature": 0.7
    }
    response = requests.post(DEEPSEEK_API_URL, headers=headers, json=payload)
    print(f"Hello, {response}")

    if "choices" not in response:
        raise ValueError(f"Unexpected response format: {response}")
    return response.json()["choices"][0]["message"]["content"]