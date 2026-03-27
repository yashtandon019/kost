package explainer

import (
	"context"
	"fmt"

	"github.com/yashtandon019/kost/pkg/detector"
)

// OllamaExplainer generates explanations using a local Ollama instance.
type OllamaExplainer struct {
	// Endpoint is the Ollama API endpoint (e.g., "http://localhost:11434").
	Endpoint string

	// Model is the model to use (e.g., "llama3", "mistral").
	Model string
}

// NewOllamaExplainer creates a new OllamaExplainer.
func NewOllamaExplainer(endpoint, model string) *OllamaExplainer {
	return &OllamaExplainer{
		Endpoint: endpoint,
		Model:    model,
	}
}

// Explain generates a human-readable explanation using a local Ollama model.
func (e *OllamaExplainer) Explain(_ context.Context, _ detector.Anomaly) (Explanation, error) {
	// TODO: Implement Ollama API integration.
	// 1. Build a prompt with anomaly data + recent K8s events
	// 2. Call the Ollama /api/generate endpoint
	// 3. Parse the response into an Explanation
	return Explanation{}, fmt.Errorf("ollama explainer not yet implemented")
}
