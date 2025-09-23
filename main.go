package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"time"

)

var (
	qdrantHost = getEnv("QDRANT_HOST", "127.0.0.1")
	qdrantPort = getEnv("QDRANT_PORT", "6333")
	collection = getEnv("QDRANT_COLLECTION", "local_rag")
)

func getEnv(key, d string) string {
	v := os.Getenv(key)
	if v == "" {
		return d
	}
	return v
}

type ChatRequest struct {
	Query string `json:"query"`
}

type QdrantSearchRequest struct {
	// Use minimal request structure by using http client instead of go-client lib for simplicity
}

func main() {
	http.HandleFunc("/chat", chatHandler)
	addr := ":8080"
	log.Println("listening on", addr)
	log.Fatal(http.ListenAndServe(addr, nil))
}

func chatHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	if r.Method != http.MethodPost {
		http.Error(w, "method must be POST", http.StatusMethodNotAllowed)
		return
	}
	var req ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request: "+err.Error(), http.StatusBadRequest)
		return
	}
	// 1) embed the query (call ollama embed)
	embedding, err := embedText(req.Query)
	if err != nil {
		http.Error(w, "failed to embed: "+err.Error(), http.StatusInternalServerError)
		print("%s",err)
	}

	// 2) query qdrant with vector search
	topk := 5
	hits, err := qdrantSearch(embedding, topk)
	if err != nil {
		http.Error(w, "qdrant search failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// 3) build prompt including top hits
	prompt := buildPrompt(req.Query, hits)

	// 4) call Ollama generate to produce final answer
	answer, err := generateFromOllama(prompt)
	if err != nil {
		http.Error(w, "generation failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	resp := map[string]any{"answer": answer, "sources": hits}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)

	_ = ctx
}

type Hit struct {
	ID     string `json:"id"`
	Score  float32 `json:"score"`
	Text   string `json:"text"`
	Source string `json:"source"`
}

func qdrantSearch(vector []float32, topk int) ([]Hit, error) {
	// Use HTTP to query Qdrant's search API
	url := fmt.Sprintf("http://%s:%s/collections/%s/points/search", qdrantHost, qdrantPort, collection)
	body := map[string]any{
		"vector": vector,
		"limit":  topk,
		"with_payload": true,
		"with_vector":  false,
	}
	bs, _ := json.Marshal(body)
	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Post(url, "application/json", bytes.NewReader(bs))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("qdrant returned %d: %s", resp.StatusCode, string(b))
	}
	var out struct {
		Result []struct {
			Id      any                    `json:"id"`
			Score   float32                `json:"score"`
			Payload map[string]interface{} `json:"payload"`
		} `json:"result"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	hits := make([]Hit, 0, len(out.Result))
	for _, r := range out.Result {
		text := ""
		src := ""
		if t, ok := r.Payload["text"].(string); ok {
			text = t
		}
		if s, ok := r.Payload["source"].(string); ok {
			src = s
		}
		idStr := fmt.Sprintf("%v", r.Id)
		hits = append(hits, Hit{ID: idStr, Score: r.Score, Text: text, Source: src})
	}
	return hits, nil
}

func buildPrompt(query string, hits []Hit) string {
	buf := &bytes.Buffer{}
	fmt.Fprintf(buf, "You are a helpful assistant. Use the following context to answer the question.\n\n")
	for i, h := range hits {
		fmt.Fprintf(buf, "Context %d (source=%s):\n%s\n\n", i+1, h.Source, h.Text)
	}
	fmt.Fprintf(buf, "Question: %s\nAnswer:", query)
	return buf.String()
}

func embedText(text string) ([]float32, error) {
	// Call ollama embed CLI synchronously and parse JSON (similar approach as ingest.py)
	cmd := exec.Command("ollama", "embed", "nomic-embed-text", text)
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	// Try to parse JSON or whitespace floats
	var parsed any
	if err := json.Unmarshal(out, &parsed); err == nil {
		switch v := parsed.(type) {
		case map[string]any:
			if emb, ok := v["embedding"].([]any); ok {
				return toFloat32Slice(emb), nil
			}
		case []any:
			return toFloat32Slice(v), nil
		}
	}
	// fallback: whitespace parse
	var floats []float32
	var f float64
	for _, tok := range bytes.Fields(out) {
		if _, err := fmt.Sscan(string(tok), &f); err == nil {
			floats = append(floats, float32(f))
		}
	}
	return floats, nil
}

func toFloat32Slice(arr []any) []float32 {
	res := make([]float32, 0, len(arr))
	for _, x := range arr {
		if f, ok := x.(float64); ok {
			res = append(res, float32(f))
		}
	}
	return res
}

func generateFromOllama(prompt string) (string, error) {
	// Use ollama CLI generate (model gemma:2b or similar)
	cmd := exec.Command("ollama", "run", "gemma:2b", "--prompt", prompt)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return "", err
	}
	return out.String(), nil
}
