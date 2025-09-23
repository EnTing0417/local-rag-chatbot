package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
)

// Request structure for /chat
type ChatRequest struct {
	Query string `json:"query"`
	Model string `json:"model"`
}

// Response structure
type ChatResponse struct {
	Answer string `json:"answer"`
}

// -------------------------
// Replace this with your actual LLM call (Ollama, etc.)
// For now it just echoes the query + model
func runLLMQuery(query string, model string) string {
	return fmt.Sprintf("[Model: %s] Response to: %s", model, query)
}

// -------------------------
// Example function to query Qdrant
func queryQdrant(query string) ([]string, error) {
	qdrantURL := "http://localhost:6333/collections/your_collection_name/points/search"

	// Example payload
	payload := map[string]interface{}{
		"vector": []float32{0.0}, // Replace with actual embedding
		"limit":  5,
	}

	payloadBytes, _ := json.Marshal(payload)
	resp, err := http.Post(qdrantURL, "application/json", bytes.NewBuffer(payloadBytes))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := ioutil.ReadAll(resp.Body)
	var searchResp QdrantSearchResponse
	err = json.Unmarshal(body, &searchResp)
	if err != nil {
		return nil, err
	}

	results := []string{}
	for _, r := range searchResp.Result {
		if text, ok := r.Payload["text"].(string); ok {
			results = append(results, text)
		}
	}
	return results, nil
}

// -------------------------
// HTTP Handler for /chat
func chatHandler(w http.ResponseWriter, r *http.Request) {
	var req ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	model := req.Model
	if model == "" {
		model = "gemma:2b" // default
	}

	// Optional: query Qdrant first
	// snippets, err := queryQdrant(req.Query)
	// if err != nil {
	//     log.Println("Qdrant error:", err)
	// }

	// Call LLM (replace with actual embedding + LLM query)
	answer := runLLMQuery(req.Query, model)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(ChatResponse{Answer: answer})
}

// -------------------------
func main() {
	port := "8080"
	if p := os.Getenv("PORT"); p != "" {
		port = p
	}

	http.HandleFunc("/chat", chatHandler)

	log.Println("Starting server on port", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal("Server failed:", err)
	}
}
