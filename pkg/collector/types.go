// Package collector provides interfaces and types for collecting
// Kubernetes resource usage metrics from a cluster.
package collector

import "time"

// ResourceSnapshot represents a point-in-time capture of resource usage
// for a single namespace.
type ResourceSnapshot struct {
	// CPUMillicores is the total CPU usage in millicores.
	CPUMillicores int64 `json:"cpuMillicores"`

	// MemoryBytes is the total memory usage in bytes.
	MemoryBytes int64 `json:"memoryBytes"`

	// GPUCount is the number of GPUs allocated.
	GPUCount int `json:"gpuCount"`

	// PodCount is the number of running pods.
	PodCount int `json:"podCount"`

	// Timestamp is when this snapshot was taken.
	Timestamp time.Time `json:"timestamp"`
}

// NamespaceUsage represents the current resource usage and estimated cost
// for a single Kubernetes namespace.
type NamespaceUsage struct {
	// Namespace is the Kubernetes namespace name.
	Namespace string `json:"namespace"`

	// Snapshots contains recent resource usage snapshots.
	Snapshots []ResourceSnapshot `json:"snapshots"`

	// EstimatedCostPerHour is the estimated hourly cost based on resource usage.
	EstimatedCostPerHour float64 `json:"estimatedCostPerHour"`
}
