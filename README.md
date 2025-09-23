Prereqs:
- Docker Desktop (to run Qdrant)
- Ollama installed and models pulled: `ollama pull nomic-embed-text` and `ollama pull gemma:2b`
- Python 3.10+, pip install requirements: `pip install qdrant-client`
- Go 1.20+

Steps:
1. Start qdrant: docker compose up -d
2. Pull ollama models:
   ollama pull nomic-embed-text
   ollama pull gemma:2b
3. Create docs/ and add .txt files
4. Ingest: python ingest.py
5. Run server: go run main.go
6. Query:
   curl -X POST http://localhost:8080/chat -H "Content-Type: application/json" -d '{"query":"What is in example.txt?"}'
