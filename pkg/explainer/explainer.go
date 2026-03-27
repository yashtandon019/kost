// Package explainer provides a pluggable interface for generating
// human-readable explanations of cost anomalies using AI/LLM backends.
package explainer

import (
	"context"

	"github.com/yashtandon019/kost/pkg/detector"
)

// Explanation is the human-readable output produced by an Explainer.
type Explanation struct {
	// Summary is a short one-line description of the anomaly.
	Summary string `json:"summary"`

	// Details is a longer explanation of what happened and why.
	Details string `json:"details"`

	// SuggestedActions are actionable recommendations to resolve the anomaly.
	SuggestedActions []string `json:"suggestedActions,omitempty"`

	// Provider identifies which explainer backend produced this explanation.
	Provider string `json:"provider"`
}

// Explainer defines the interface for generating human-readable
// explanations of cost anomalies.
type Explainer interface {
	// Explain generates a human-readable explanation for the given anomaly.
	Explain(ctx context.Context, anomaly detector.Anomaly) (Explanation, error)
}
