package main

import "fmt"

// QueryLLM sends query + context to your model (Ollama or other LLM)
func QueryLLM(query string, context []string, model string) string {
	// TODO: Replace with real Ollama / LLM call
	combinedContext := ""
	for _, c := range context {
		combinedContext += c + "\n"
	}

	return fmt.Sprintf("[Model: %s] Answer to: %s\nContext:\n%s", model, query, combinedContext)
}
