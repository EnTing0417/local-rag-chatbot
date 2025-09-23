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
    QdrantURL      = "http://localhost:6333"
    CollectionName = "documents"
    VectorName     = "embedding" // must match ingest.py
    LimitResults   = 5
)

func main() {
    http.HandleFunc("/chat", chatHandler)
    log.Println("server on :8080")
    log.Fatal(http.ListenAndServe(":8080", nil))
}

func chatHandler(w http.ResponseWriter, r *http.Request) {
    var req struct {
        Query string `json:"query"`
    }
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, "invalid request: "+err.Error(), http.StatusBadRequest)
        return
    }

    // 🔴 Replace with actual Ollama embedding call
    queryVec, err := getEmbedding(req.Query)
    if err != nil {
        http.Error(w, "embedding error: "+err.Error(), http.StatusInternalServerError)
        return
    }

    searchBody := map[string]interface{}{
        "vector":  queryVec, 
        "limit":        LimitResults,
        "with_payload": true,
    }

    b, _ := json.Marshal(searchBody)
    qurl := fmt.Sprintf("%s/collections/%s/points/search", QdrantURL, CollectionName)
    resp, err := http.Post(qurl, "application/json", bytes.NewBuffer(b))
    if err != nil {
        http.Error(w, "qdrant request error: "+err.Error(), http.StatusInternalServerError)
        return
    }
    defer resp.Body.Close()

    bodyResp, _ := io.ReadAll(resp.Body)
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(resp.StatusCode)
    w.Write(bodyResp)
}

func getEmbedding(text string) ([]float64, error) {
    url := "http://localhost:11434/api/embeddings"
    body := map[string]interface{}{
        "model":  "nomic-embed-text", // or your preferred embedding model
        "prompt": text,
    }
    b, _ := json.Marshal(body)

    resp, err := http.Post(url, "application/json", bytes.NewBuffer(b))
    if err != nil {
        return nil, fmt.Errorf("ollama request failed: %w", err)
    }
    defer resp.Body.Close()

    var out struct {
        Embedding []float64 `json:"embedding"`
    }
    if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
        return nil, fmt.Errorf("decode error: %w", err)
    }
    return out.Embedding, nil
}
