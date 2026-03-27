package alerter

import (
	"context"
	"fmt"

	"github.com/yashtandon019/kost/pkg/explainer"
)

// EventAlerter emits Kubernetes Events for anomaly alerts.
// Events are attached to the KostConfig resource and visible via kubectl.
type EventAlerter struct {
	// Namespace where K8s Events will be created.
	Namespace string
}

// NewEventAlerter creates a new EventAlerter.
func NewEventAlerter(namespace string) *EventAlerter {
	return &EventAlerter{
		Namespace: namespace,
	}
}

// Alert emits a Kubernetes Event with the anomaly explanation.
func (a *EventAlerter) Alert(_ context.Context, _ explainer.Explanation) error {
	// TODO: Implement Kubernetes Event emission.
	// 1. Create an EventRecorder using client-go
	// 2. Emit a Warning event with the anomaly summary
	// 3. Include details in the event message
	return fmt.Errorf("event alerter not yet implemented")
}
