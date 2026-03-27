package explainer

import (
	"context"
	"fmt"

	"github.com/yashtandon019/kost/pkg/detector"
)

// ClaudeExplainer generates explanations using the Anthropic Claude API.
type ClaudeExplainer struct {
	// APIKey is the Anthropic API key.
	APIKey string

	// Model is the Claude model to use (e.g., "claude-sonnet-4-20250514").
	Model string
}

// NewClaudeExplainer creates a new ClaudeExplainer.
func NewClaudeExplainer(apiKey, model string) *ClaudeExplainer {
	return &ClaudeExplainer{
		APIKey: apiKey,
		Model:  model,
	}
}

// Explain generates a human-readable explanation using Claude.
func (e *ClaudeExplainer) Explain(_ context.Context, _ detector.Anomaly) (Explanation, error) {
	// TODO: Implement Claude API integration.
	// 1. Build a prompt with anomaly data + recent K8s events
	// 2. Call the Anthropic Messages API
	// 3. Parse the response into an Explanation
	return Explanation{}, fmt.Errorf("claude explainer not yet implemented")
}
