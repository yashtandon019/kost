package collector

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	metricsv1beta1 "k8s.io/metrics/pkg/apis/metrics/v1beta1"
	metricsclient "k8s.io/metrics/pkg/client/clientset/versioned"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/yashtandon019/kost/pkg/pricing"
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

	// Pricing is the pricing configuration used to estimate costs.
	Pricing pricing.PricingConfig

	// kubeClient is the Kubernetes core client (for pod specs).
	kubeClient kubernetes.Interface

	// metricsClient is the metrics-server API client.
	metricsClient metricsclient.Interface
}

// NewMetricsServerCollector creates a new collector that uses metrics-server.
func NewMetricsServerCollector(kubeConfigPath string, pricingCfg pricing.PricingConfig) *MetricsServerCollector {
	return &MetricsServerCollector{
		KubeConfigPath: kubeConfigPath,
		Pricing:        pricingCfg,
	}
}

// buildConfig returns a *rest.Config from kubeconfig path or in-cluster config.
func (c *MetricsServerCollector) buildConfig() (*rest.Config, error) {
	if c.KubeConfigPath != "" {
		return clientcmd.BuildConfigFromFlags("", c.KubeConfigPath)
	}

	if cfg, err := rest.InClusterConfig(); err == nil {
		return cfg, nil
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("resolving user home directory: %w", err)
	}

	kubeconfigPath := filepath.Join(homeDir, ".kube", "config")
	cfg, err := clientcmd.BuildConfigFromFlags("", kubeconfigPath)
	if err != nil {
		return nil, fmt.Errorf("building kubeconfig from %s: %w", kubeconfigPath, err)
	}
	return cfg, nil
}

// ensureClients lazily initialises the Kubernetes and metrics clients.
func (c *MetricsServerCollector) ensureClients() error {
	if c.kubeClient != nil && c.metricsClient != nil {
		return nil
	}

	cfg, err := c.buildConfig()
	if err != nil {
		return fmt.Errorf("building kubeconfig: %w", err)
	}

	kube, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return fmt.Errorf("creating kubernetes client: %w", err)
	}

	metrics, err := metricsclient.NewForConfig(cfg)
	if err != nil {
		return fmt.Errorf("creating metrics client: %w", err)
	}

	c.kubeClient = kube
	c.metricsClient = metrics
	return nil
}

// gpuResourceName is the extended resource name used by the NVIDIA device plugin.
const gpuResourceName corev1.ResourceName = "nvidia.com/gpu"

// Collect gathers resource usage from metrics-server for all namespaces.
func (c *MetricsServerCollector) Collect(ctx context.Context) ([]NamespaceUsage, error) {
	logger := log.FromContext(ctx)

	if err := c.ensureClients(); err != nil {
		return nil, err
	}

	// List all namespaces.
	nsList, err := c.kubeClient.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("listing namespaces: %w", err)
	}

	now := time.Now().UTC()
	var usages []NamespaceUsage

	for _, ns := range nsList.Items {
		usage, err := c.collectNamespace(ctx, ns.Name, now)
		if err != nil {
			logger.Error(err, "Failed to collect metrics for namespace", "namespace", ns.Name)
			continue
		}
		usages = append(usages, usage)
	}

	logger.V(1).Info("Collected namespace metrics", "count", len(usages))
	return usages, nil
}

// collectNamespace gathers metrics for a single namespace.
func (c *MetricsServerCollector) collectNamespace(
	ctx context.Context,
	namespace string,
	now time.Time,
) (NamespaceUsage, error) {
	podMetrics, err := c.metricsClient.MetricsV1beta1().PodMetricses(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return NamespaceUsage{}, fmt.Errorf("listing pod metrics: %w", err)
	}

	snapshot := aggregateMetrics(podMetrics.Items, now)

	// Detect GPU allocations from pod resource requests.
	gpuCount, err := c.countGPUs(ctx, namespace)
	if err != nil {
		// Non-fatal: log and continue without GPU data.
		log.FromContext(ctx).V(1).Info("Could not count GPUs", "namespace", namespace, "error", err)
	}
	snapshot.GPUCount = gpuCount

	cost := c.Pricing.EstimateCostFromMillis(snapshot.CPUMillicores, snapshot.MemoryBytes, snapshot.GPUCount)

	return NamespaceUsage{
		Namespace:            namespace,
		Snapshots:            []ResourceSnapshot{snapshot},
		EstimatedCostPerHour: cost,
	}, nil
}

// aggregateMetrics sums CPU and memory across all containers in the given pod metrics.
func aggregateMetrics(pods []metricsv1beta1.PodMetrics, now time.Time) ResourceSnapshot {
	var cpuTotal, memTotal int64
	for _, pod := range pods {
		for _, container := range pod.Containers {
			cpuTotal += container.Usage.Cpu().MilliValue()
			memTotal += container.Usage.Memory().Value()
		}
	}
	return ResourceSnapshot{
		CPUMillicores: cpuTotal,
		MemoryBytes:   memTotal,
		PodCount:      len(pods),
		Timestamp:     now,
	}
}

// countGPUs counts the total number of nvidia.com/gpu resources requested
// by pods in the given namespace.
func (c *MetricsServerCollector) countGPUs(ctx context.Context, namespace string) (int, error) {
	pods, err := c.kubeClient.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{
		FieldSelector: "status.phase=Running",
	})
	if err != nil {
		return 0, fmt.Errorf("listing pods: %w", err)
	}

	var total int
	for _, pod := range pods.Items {
		for _, container := range pod.Spec.Containers {
			if qty, ok := container.Resources.Requests[gpuResourceName]; ok {
				total += int(qty.Value())
			}
		}
	}
	return total, nil
}
