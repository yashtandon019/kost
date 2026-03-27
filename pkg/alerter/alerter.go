// Package alerter provides interfaces and implementations for sending
// cost anomaly alerts to various notification sinks.
package alerter

import (
	"context"

	"github.com/yashtandon019/kost/pkg/explainer"
)

// Alerter defines the interface for sending anomaly alerts.
type Alerter interface {
	// Alert sends an anomaly explanation to the configured sink.
	Alert(ctx context.Context, explanation explainer.Explanation) error
}
