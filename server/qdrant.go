package main

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
)

type QdrantSearchResponse struct {
	Result []struct {
		Payload map[string]interface{} `json:"payload"`
	} `json:"result"`
}

// QueryQdrant searches the vector DB and returns top text snippets
func QueryQdrant(query string) ([]string, error) {
	qdrantURL := "http://qdrant:6333/collections/your_collection_name/points/search"

	payload := map[string]interface{}{
		"vector": []float32{0.0}, // TODO: replace with real embedding
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
	if err := json.Unmarshal(body, &searchResp); err != nil {
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
