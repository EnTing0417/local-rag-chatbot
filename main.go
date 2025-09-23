package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
)

const (
	QdrantURL      = "http://localhost:6333"
	CollectionName = "documents"
	LimitResults   = 5
)

var (
	OLLAMA_EMBED_URL = getEnv("OLLAMA_URL", "http://localhost:11434/api/embeddings")
	EMBED_MODEL      = getEnv("EMBED_MODEL", "nomic-embed-text")
)

func getEnv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func main() {
	http.HandleFunc("/chat", chatHandler)
	log.Println("server on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

type chatReq struct {
	Query string `json:"query"`
}

func chatHandler(w http.ResponseWriter, r *http.Request) {
	var req chatReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}
	if req.Query == "" {
		http.Error(w, "empty query", http.StatusBadRequest)
		return
	}

	vec, err := getEmbedding(req.Query)
	if err != nil {
		http.Error(w, "embedding error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Search body for UNNAMED vector mode:
	searchBody := map[string]interface{}{
		"vector":        vec,
		"limit":         LimitResults,
		"with_payload":  true,
		"with_vectors":  false,
	}

	searchBytes, _ := json.Marshal(searchBody)
	qurl := fmt.Sprintf("%s/collections/%s/points/search", QdrantURL, CollectionName)
	resp, err := http.Post(qurl, "application/json", bytes.NewBuffer(searchBytes))
	if err != nil {
		http.Error(w, "qdrant request error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()
	bodyResp, _ := io.ReadAll(resp.Body)

	// Forward Qdrant response and status code
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(resp.StatusCode)
	w.Write(bodyResp)
}

func getEmbedding(text string) ([]float64, error) {
	payload := map[string]interface{}{
		"model":  EMBED_MODEL,
		"prompt": text,
	}
	b, _ := json.Marshal(payload)
	resp, err := http.Post(OLLAMA_EMBED_URL, "application/json", bytes.NewBuffer(b))
	if err != nil {
		return nil, fmt.Errorf("ollama request failed: %w", err)
	}
	defer resp.Body.Close()
	var out struct {
		Embedding []float64 `json:"embedding"`
		// some models return embeddings: [...]
		Embeddings [][]float64 `json:"embeddings"`
	}
	dec := json.NewDecoder(resp.Body)
	if err := dec.Decode(&out); err != nil {
		return nil, fmt.Errorf("decode embedding response: %w", err)
	}
	if len(out.Embedding) > 0 {
		return out.Embedding, nil
	}
	if len(out.Embeddings) > 0 {
		return out.Embeddings[0], nil
	}
	return nil, fmt.Errorf("embedding not found in ollama response")
}
