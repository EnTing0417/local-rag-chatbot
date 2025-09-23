package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
)

// QueryLLM sends a query and context to the Ollama LLM and returns the response.
func QueryLLM(query string, context []string, model string) (string, error) {
	combinedContext := ""
	for _, c := range context {
		combinedContext += c + "\n"
	}

	payload := map[string]interface{}{
		"model":      model,
		"prompt":     fmt.Sprintf("Context:\n%s\nQuestion:\n%s", combinedContext, query),
		"max_tokens": 512,
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	resp, err := http.Post("http://localhost:11434/completion", "application/json", bytes.NewReader(payloadBytes))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(resp.Body)
		return "", fmt.Errorf("Ollama API error: %s", string(body))
	}

	var result struct {
		Completion string `json:"completion"`
	}
	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		return "", err
	}

	return result.Completion, nil
}
