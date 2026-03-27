package explainer

import (
	"context"
	"fmt"

	"github.com/yashtandon019/kost/pkg/detector"
)

// OpenAIExplainer generates explanations using the OpenAI API.
type OpenAIExplainer struct {
	// APIKey is the OpenAI API key.
	APIKey string

	// Model is the model to use (e.g., "gpt-4o").
	Model string
}

// NewOpenAIExplainer creates a new OpenAIExplainer.
func NewOpenAIExplainer(apiKey, model string) *OpenAIExplainer {
	return &OpenAIExplainer{
		APIKey: apiKey,
		Model:  model,
	}
}

// Explain generates a human-readable explanation using OpenAI.
func (e *OpenAIExplainer) Explain(_ context.Context, _ detector.Anomaly) (Explanation, error) {
	// TODO: Implement OpenAI API integration.
	// 1. Build a prompt with anomaly data + recent K8s events
	// 2. Call the OpenAI Chat Completions API
	// 3. Parse the response into an Explanation
	return Explanation{}, fmt.Errorf("openai explainer not yet implemented")
}
