import requests
import os
import time

OLLAMA_HOST = os.getenv("OLLAMA_HOST", "http://ollama:11434")

def retrieve_context(query):
    # Placeholder for RAG logic: vector search, document retrieval, etc.
    return "Relevant context from your documents."


def query_ollama(prompt, retries=10, delay=5):
    for attempt in range(retries):
        try:
            response = requests.post(
                f"{OLLAMA_HOST}/api/generate",
                json={"model": "mistral", "prompt": prompt, "stream": False}
            )

            if response.status_code == 404:
                print(f"Ollama not ready (attempt {attempt+1}): 404 Not Found")
                time.sleep(delay)
                continue  # Try again

            return response.json().get("response", "")

        except requests.exceptions.RequestException as e:
            print(f"Ollama not ready (attempt {attempt+1}): {e}")
            time.sleep(delay)

    return "❌ Ollama is not responding after multiple attempts."
