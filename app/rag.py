import requests
import os

OLLAMA_HOST = os.getenv("OLLAMA_HOST", "http://ollama:11434")

def retrieve_context(query):
    # Placeholder for RAG logic: vector search, document retrieval, etc.
    return "Relevant context from your documents."

def query_ollama(prompt):
    response = requests.post(
        f"{OLLAMA_HOST}/api/generate",
        json={"model": "mistral", "prompt": prompt,"stream": False}
    )
    print(response.json())
    return response.json().get("response", "")