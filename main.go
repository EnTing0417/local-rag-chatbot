package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
)

const (
	QdrantURL        = "http://localhost:6333"
	OllamaURL        = "http://localhost:11434/api"
	QdrantCollection = "docs"
	TopK             = 3
	EmbeddingModel   = "nomic-embed-text"
	GenModel         = "gemma:2b"
)

type ChatRequest struct {
	Query string `json:"query"`
}

type ChatResponse struct {
	Answer string `json:"answer"`
}

func getEmbedding(text string) ([]float32, error) {
	type EmbeddingRequest struct {
		Model  string `json:"model"`
		Prompt string `json:"prompt"`
	}
	type EmbeddingResponse struct {
		Embedding []float32 `json:"embedding"`
	}

	reqBody, _ := json.Marshal(EmbeddingRequest{Model: EmbeddingModel, Prompt: text})
	resp, err := http.Post(OllamaURL+"/embeddings", "application/json", bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("ollama embedding error: %s", string(b))
	}

	var er EmbeddingResponse
	if err := json.NewDecoder(resp.Body).Decode(&er); err != nil {
		return nil, err
	}
	return er.Embedding, nil
}

func searchQdrant(vector []float32, topK int) (string, error) {
	type QdrantSearchRequest struct {
		Vector      []float32 `json:"vector"`
		Limit       int       `json:"limit"`
		WithPayload bool      `json:"with_payload"`
	}
	type QdrantSearchResponse struct {
		Result []struct {
			Payload map[string]interface{} `json:"payload"`
		} `json:"result"`
	}

	body, _ := json.Marshal(QdrantSearchRequest{Vector: vector, Limit: topK, WithPayload: true})
	resp, err := http.Post(QdrantURL+"/collections/"+QdrantCollection+"/points/search", "application/json", bytes.NewBuffer(body))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("qdrant search error: %s", string(b))
	}

	var qresp QdrantSearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&qresp); err != nil {
		return "", err
	}

	context := ""
	for _, r := range qresp.Result {
		if txt, ok := r.Payload["text"].(string); ok {
			context += txt + "\n"
		}
	}
	return context, nil
}

func buildPrompt(context, query string) string {
	return fmt.Sprintf("Use the following context to answer the question.\nContext:\n%s\n\nQuestion: %s\nAnswer:", context, query)
}

func callOllama(prompt string) (string, error) {
	type OllamaRequest struct {
		Model  string `json:"model"`
		Prompt string `json:"prompt"`
	}
	type OllamaResponse struct {
		Response string `json:"response"`
		Done     bool   `json:"done"`
	}

	reqBody, _ := json.Marshal(OllamaRequest{Model: GenModel, Prompt: prompt})
	resp, err := http.Post(OllamaURL+"/generate", "application/json", bytes.NewBuffer(reqBody))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var answer string
	dec := json.NewDecoder(resp.Body)
	for dec.More() {
		var or OllamaResponse
		if err := dec.Decode(&or); err != nil {
			break
		}
		answer += or.Response
		if or.Done {
			break
		}
	}
	return answer, nil
}

func chatHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON body", http.StatusBadRequest)
		return
	}

	vec, err := getEmbedding(req.Query)
	if err != nil {
		http.Error(w, "embedding error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	context, err := searchQdrant(vec, TopK)
	if err != nil {
		http.Error(w, "qdrant error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	prompt := buildPrompt(context, req.Query)
	answer, err := callOllama(prompt)
	if err != nil {
		http.Error(w, "ollama error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(ChatResponse{Answer: answer})
}

func main() {
	http.HandleFunc("/chat", chatHandler)
	fmt.Println("Server listening on http://localhost:8080/chat")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
