package collector

import (
	"context"
	"fmt"
)

// Collector defines the interface for collecting resource usage metrics
// from a Kubernetes cluster.
type Collector interface {
	// Collect gathers current resource usage for all namespaces.
	Collect(ctx context.Context) ([]NamespaceUsage, error)
}

// MetricsServerCollector collects resource usage from the Kubernetes
// metrics-server API.
type MetricsServerCollector struct {
	// KubeConfigPath is the path to the kubeconfig file.
	// If empty, in-cluster config is used.
	KubeConfigPath string
}

// NewMetricsServerCollector creates a new collector that uses metrics-server.
func NewMetricsServerCollector(kubeConfigPath string) *MetricsServerCollector {
	return &MetricsServerCollector{
		KubeConfigPath: kubeConfigPath,
	}
}

// Collect gathers resource usage from metrics-server for all namespaces.
func (c *MetricsServerCollector) Collect(_ context.Context) ([]NamespaceUsage, error) {
	// TODO: Implement metrics-server collection using client-go.
	// 1. List all namespaces
	// 2. For each namespace, query pod metrics from metrics.k8s.io/v1beta1
	// 3. Aggregate CPU, memory, GPU, and pod counts
	// 4. Return NamespaceUsage with resource snapshots
	return nil, fmt.Errorf("metrics-server collector not yet implemented")
}
