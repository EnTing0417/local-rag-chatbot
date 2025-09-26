## Prerequisites
- Docker installed
- Docker Compose installed
- At least 8GB RAM (more for larger models)
(Optional) GPU + CUDA for acceleration


```bash
# Clone repository
git clone https://github.com/EnTing0417/local-rag-chatbot.git
cd local-rag-chatbot

# Build and start containers
docker compose up --build
```

This will spin up:
    - Ollama service on port 11434
    - Chatbot UI on port 8000

After starting the stack:
Open the web UI at 👉 http://localhost:8000