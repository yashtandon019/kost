package collector

import (
	"context"
	"testing"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	kubefake "k8s.io/client-go/kubernetes/fake"
	clienttesting "k8s.io/client-go/testing"
	metricsv1beta1 "k8s.io/metrics/pkg/apis/metrics/v1beta1"
	metricsfake "k8s.io/metrics/pkg/client/clientset/versioned/fake"

	"github.com/yashtandon019/kost/pkg/pricing"
)

func newFakeMetricsClient(namespacedMetrics map[string][]metricsv1beta1.PodMetrics) *metricsfake.Clientset {
	client := metricsfake.NewSimpleClientset()
	client.PrependReactor("list", "pods", func(action clienttesting.Action) (bool, runtime.Object, error) {
		listAction, ok := action.(clienttesting.ListAction)
		if !ok {
			return false, nil, nil
		}

		items := namespacedMetrics[listAction.GetNamespace()]
		return true, &metricsv1beta1.PodMetricsList{
			Items: append([]metricsv1beta1.PodMetrics(nil), items...),
		}, nil
	})
	return client
}

func TestCollect_SingleNamespace(t *testing.T) {
	kubeClient := kubefake.NewSimpleClientset(
		&corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "default"}},
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{Name: "pod-1", Namespace: "default"},
			Status:     corev1.PodStatus{Phase: corev1.PodRunning},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{{
					Name: "app",
					Resources: corev1.ResourceRequirements{
						Requests: corev1.ResourceList{},
					},
				}},
			},
		},
	)

	metricsClient := newFakeMetricsClient(map[string][]metricsv1beta1.PodMetrics{
		"default": {{
			ObjectMeta: metav1.ObjectMeta{Name: "pod-1", Namespace: "default"},
			Containers: []metricsv1beta1.ContainerMetrics{{
				Name: "app",
				Usage: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("250m"),
					corev1.ResourceMemory: resource.MustParse("128Mi"),
				},
			}},
		}},
	})

	c := &MetricsServerCollector{
		Pricing:       pricing.DefaultPricing(),
		kubeClient:    kubeClient,
		metricsClient: metricsClient,
	}

	usages, err := c.Collect(context.Background())
	if err != nil {
		t.Fatalf("Collect returned error: %v", err)
	}

	if len(usages) != 1 {
		t.Fatalf("Expected 1 namespace usage, got %d", len(usages))
	}

	u := usages[0]
	if u.Namespace != "default" {
		t.Errorf("Expected namespace 'default', got %q", u.Namespace)
	}

	if len(u.Snapshots) != 1 {
		t.Fatalf("Expected 1 snapshot, got %d", len(u.Snapshots))
	}

	snap := u.Snapshots[0]
	if snap.CPUMillicores != 250 {
		t.Errorf("Expected 250 CPU millicores, got %d", snap.CPUMillicores)
	}

	expectedMem := int64(128 * 1024 * 1024)
	if snap.MemoryBytes != expectedMem {
		t.Errorf("Expected %d memory bytes, got %d", expectedMem, snap.MemoryBytes)
	}

	if snap.PodCount != 1 {
		t.Errorf("Expected pod count 1, got %d", snap.PodCount)
	}

	if u.EstimatedCostPerHour <= 0 {
		t.Errorf("Expected positive estimated cost, got %f", u.EstimatedCostPerHour)
	}
}

func TestCollect_MultipleNamespaces(t *testing.T) {
	kubeClient := kubefake.NewSimpleClientset(
		&corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "ns-a"}},
		&corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "ns-b"}},
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{Name: "pod-a", Namespace: "ns-a"},
			Status:     corev1.PodStatus{Phase: corev1.PodRunning},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{{Name: "c"}},
			},
		},
	)

	metricsClient := newFakeMetricsClient(map[string][]metricsv1beta1.PodMetrics{
		"ns-a": {{
			ObjectMeta: metav1.ObjectMeta{Name: "pod-a", Namespace: "ns-a"},
			Containers: []metricsv1beta1.ContainerMetrics{{
				Name: "c",
				Usage: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("500m"),
					corev1.ResourceMemory: resource.MustParse("256Mi"),
				},
			}},
		}},
	})

	c := &MetricsServerCollector{
		Pricing:       pricing.DefaultPricing(),
		kubeClient:    kubeClient,
		metricsClient: metricsClient,
	}

	usages, err := c.Collect(context.Background())
	if err != nil {
		t.Fatalf("Collect returned error: %v", err)
	}

	if len(usages) != 2 {
		t.Fatalf("Expected 2 namespace usages, got %d", len(usages))
	}

	// Find ns-a usage.
	var nsA *NamespaceUsage
	for i := range usages {
		if usages[i].Namespace == "ns-a" {
			nsA = &usages[i]
			break
		}
	}
	if nsA == nil {
		t.Fatal("Expected to find namespace ns-a in results")
	}

	snap := nsA.Snapshots[0]
	if snap.CPUMillicores != 500 {
		t.Errorf("Expected 500 CPU millicores for ns-a, got %d", snap.CPUMillicores)
	}
	if snap.PodCount != 1 {
		t.Errorf("Expected pod count 1 for ns-a, got %d", snap.PodCount)
	}
}

func TestCollect_GPUDetection(t *testing.T) {
	gpuQty := resource.MustParse("2")

	kubeClient := kubefake.NewSimpleClientset(
		&corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "gpu-ns"}},
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{Name: "gpu-pod", Namespace: "gpu-ns"},
			Status:     corev1.PodStatus{Phase: corev1.PodRunning},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{{
					Name: "trainer",
					Resources: corev1.ResourceRequirements{
						Requests: corev1.ResourceList{
							gpuResourceName: gpuQty,
						},
					},
				}},
			},
		},
	)

	metricsClient := newFakeMetricsClient(map[string][]metricsv1beta1.PodMetrics{
		"gpu-ns": {{
			ObjectMeta: metav1.ObjectMeta{Name: "gpu-pod", Namespace: "gpu-ns"},
			Containers: []metricsv1beta1.ContainerMetrics{{
				Name: "trainer",
				Usage: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("1000m"),
					corev1.ResourceMemory: resource.MustParse("4Gi"),
				},
			}},
		}},
	})

	c := &MetricsServerCollector{
		Pricing:       pricing.DefaultPricing(),
		kubeClient:    kubeClient,
		metricsClient: metricsClient,
	}

	usages, err := c.Collect(context.Background())
	if err != nil {
		t.Fatalf("Collect returned error: %v", err)
	}

	if len(usages) != 1 {
		t.Fatalf("Expected 1 usage, got %d", len(usages))
	}

	snap := usages[0].Snapshots[0]
	if snap.GPUCount != 2 {
		t.Errorf("Expected GPU count 2, got %d", snap.GPUCount)
	}

	// Cost should include GPU pricing.
	expectedMinCost := pricing.DefaultPricing().GPUPerHour * 2
	if usages[0].EstimatedCostPerHour < expectedMinCost {
		t.Errorf("Expected cost >= %f (GPU component alone), got %f", expectedMinCost, usages[0].EstimatedCostPerHour)
	}
}

func TestCollect_EmptyNamespace(t *testing.T) {
	kubeClient := kubefake.NewSimpleClientset(
		&corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "empty"}},
	)

	metricsClient := newFakeMetricsClient(nil)

	c := &MetricsServerCollector{
		Pricing:       pricing.DefaultPricing(),
		kubeClient:    kubeClient,
		metricsClient: metricsClient,
	}

	usages, err := c.Collect(context.Background())
	if err != nil {
		t.Fatalf("Collect returned error: %v", err)
	}

	if len(usages) != 1 {
		t.Fatalf("Expected 1 usage, got %d", len(usages))
	}

	snap := usages[0].Snapshots[0]
	if snap.CPUMillicores != 0 || snap.MemoryBytes != 0 || snap.PodCount != 0 {
		t.Errorf("Expected zero resource usage for empty namespace, got cpu=%d mem=%d pods=%d",
			snap.CPUMillicores, snap.MemoryBytes, snap.PodCount)
	}

	if usages[0].EstimatedCostPerHour != 0 {
		t.Errorf("Expected zero cost for empty namespace, got %f", usages[0].EstimatedCostPerHour)
	}
}

func TestCollect_MultipleContainersAggregated(t *testing.T) {
	kubeClient := kubefake.NewSimpleClientset(
		&corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "multi"}},
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{Name: "pod-1", Namespace: "multi"},
			Status:     corev1.PodStatus{Phase: corev1.PodRunning},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{Name: "sidecar"},
					{Name: "app"},
				},
			},
		},
	)

	metricsClient := newFakeMetricsClient(map[string][]metricsv1beta1.PodMetrics{
		"multi": {{
			ObjectMeta: metav1.ObjectMeta{Name: "pod-1", Namespace: "multi"},
			Containers: []metricsv1beta1.ContainerMetrics{
				{
					Name: "sidecar",
					Usage: corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse("100m"),
						corev1.ResourceMemory: resource.MustParse("64Mi"),
					},
				},
				{
					Name: "app",
					Usage: corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse("400m"),
						corev1.ResourceMemory: resource.MustParse("256Mi"),
					},
				},
			},
		}},
	})

	c := &MetricsServerCollector{
		Pricing:       pricing.DefaultPricing(),
		kubeClient:    kubeClient,
		metricsClient: metricsClient,
	}

	usages, err := c.Collect(context.Background())
	if err != nil {
		t.Fatalf("Collect returned error: %v", err)
	}

	snap := usages[0].Snapshots[0]
	if snap.CPUMillicores != 500 {
		t.Errorf("Expected 500 CPU millicores (100+400), got %d", snap.CPUMillicores)
	}

	expectedMem := int64((64 + 256) * 1024 * 1024)
	if snap.MemoryBytes != expectedMem {
		t.Errorf("Expected %d memory bytes (64Mi+256Mi), got %d", expectedMem, snap.MemoryBytes)
	}
}

func TestAggregateMetrics_NoPods(t *testing.T) {
	snap := aggregateMetrics(nil, metav1.Now().Time)
	if snap.CPUMillicores != 0 || snap.MemoryBytes != 0 || snap.PodCount != 0 {
		t.Errorf("Expected zero snapshot for nil pods, got %+v", snap)
	}
}
