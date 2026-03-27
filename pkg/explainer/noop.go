package explainer

import (
	"context"
	"fmt"

	"github.com/yashtandon019/kost/pkg/detector"
)

// NoopExplainer generates basic explanations from raw anomaly data
// without calling any external LLM API.
type NoopExplainer struct{}

// NewNoopExplainer creates a new NoopExplainer.
func NewNoopExplainer() *NoopExplainer {
	return &NoopExplainer{}
}

// Explain returns a formatted explanation built from the anomaly's raw fields.
func (e *NoopExplainer) Explain(_ context.Context, anomaly detector.Anomaly) (Explanation, error) {
	summary := fmt.Sprintf(
		"Cost anomaly in namespace %q: $%.2f/hr (baseline: $%.2f/hr, +%.0f%%)",
		anomaly.Namespace,
		anomaly.CurrentCost,
		anomaly.BaselineCost,
		anomaly.DeviationPercent,
	)

	details := fmt.Sprintf(
		"Namespace %q is currently costing $%.2f/hr, which is %.0f%% above the baseline of $%.2f/hr. "+
			"Severity: %s. Detected at %s.",
		anomaly.Namespace,
		anomaly.CurrentCost,
		anomaly.DeviationPercent,
		anomaly.BaselineCost,
		anomaly.Severity,
		anomaly.DetectedAt.Format("2006-01-02 15:04:05"),
	)

	actions := []string{
		fmt.Sprintf("Investigate recent deployments in namespace %q", anomaly.Namespace),
		"Check for unexpected scaling events or resource-heavy workloads",
		"Consider adding ResourceQuotas if not already configured",
	}

	return Explanation{
		Summary:          summary,
		Details:          details,
		SuggestedActions: actions,
		Provider:         "noop",
	}, nil
}
