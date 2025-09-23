package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/qdrant/go-client/qdrant"
	"github.com/qdrant/go-client/qdrant/models"
)

var qdrantClient *qdrant.Client

func init() {
	var err error
	qdrantClient, err = qdrant.NewClient(qdrant.WithAddress("localhost:6333"))
	if err != nil {
		log.Fatal(err)
	}
}

func getContextFromQdrant(query string, topK int) ([]string, error) {
	// Step 1: Embed the query using Ollama embedding model
	resp, err := http.Post(
		"http://localhost:11434/embed",
		"application/json",
		bytes.NewReader([]byte(fmt.Sprintf(`{"model":"nomic-embed-text","text":"%s"}`, query))),
	)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var embedResp struct {
		Embedding []float32 `json:"embedding"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&embedResp); err != nil {
		return nil, err
	}

	// Step 2: Search Qdrant
	searchResp, err := qdrantClient.Search(context.Background(), &models.SearchPoints{
		CollectionName: "documents",
		Vector:         embedResp.Embedding,
		Limit:          int64(topK),
	})
	if err != nil {
		return nil, err
	}

	// Step 3: Extract text from payload
	contexts := []string{}
	for _, hit := range searchResp.Result {
		if text, ok := hit.Payload["text"].(string); ok {
			contexts = append(contexts, text)
		}
	}

	return contexts, nil
}

func queryHandler(w http.ResponseWriter, r *http.Request) {
	type Request struct {
		Query string `json:"query"`
	}
	type Response struct {
		Answer string `json:"answer"`
	}

	var req Request
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	contexts, err := getContextFromQdrant(req.Query, 3)
	if err != nil {
		http.Error(w, "Failed to get context: "+err.Error(), http.StatusInternalServerError)
		return
	}

	answer, err := QueryLLM(req.Query, contexts, "gemma:2b")
	if err != nil {
		http.Error(w, "LLM query failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(Response{Answer: answer})
}

func main() {
	http.HandleFunc("/query", queryHandler)
	fmt.Println("Server running on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
