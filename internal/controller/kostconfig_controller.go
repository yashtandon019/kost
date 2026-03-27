/*
Copyright 2026.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"context"
	"fmt"
	"sync"
	"time"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	kostv1alpha1 "github.com/yashtandon019/kost/api/v1alpha1"
	"github.com/yashtandon019/kost/pkg/collector"
	"github.com/yashtandon019/kost/pkg/detector"
	"github.com/yashtandon019/kost/pkg/explainer"
	"github.com/yashtandon019/kost/pkg/pricing"
)

const (
	// Condition types for KostConfig status.
	conditionTypeAvailable = "Available"
	conditionTypeDegraded  = "Degraded"

	defaultPollingInterval = 5 * time.Minute
)

// KostConfigReconciler reconciles a KostConfig object.
type KostConfigReconciler struct {
	client.Client
	Scheme *runtime.Scheme

	// detectors holds per-KostConfig detector instances so baselines
	// persist across reconciliation cycles.
	detectorsMu sync.Mutex
	detectors   map[string]*detector.Detector
}

// NewKostConfigReconciler creates a new reconciler with initialized internal state.
func NewKostConfigReconciler(c client.Client, scheme *runtime.Scheme) *KostConfigReconciler {
	return &KostConfigReconciler{
		Client:    c,
		Scheme:    scheme,
		detectors: make(map[string]*detector.Detector),
	}
}

// +kubebuilder:rbac:groups=kost.kost.dev,resources=kostconfigs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=kost.kost.dev,resources=kostconfigs/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=kost.kost.dev,resources=kostconfigs/finalizers,verbs=update
// +kubebuilder:rbac:groups="",resources=pods,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=namespaces,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=events,verbs=create;patch
// +kubebuilder:rbac:groups=metrics.k8s.io,resources=pods,verbs=get;list

// Reconcile runs the collect → detect → explain → alert pipeline.
func (r *KostConfigReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	// 1. Fetch KostConfig CR.
	var config kostv1alpha1.KostConfig
	if err := r.Get(ctx, req.NamespacedName, &config); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// 2. Parse polling interval.
	requeueAfter := defaultPollingInterval
	if config.Spec.PollingInterval != "" {
		parsed, err := time.ParseDuration(config.Spec.PollingInterval)
		if err != nil {
			log.Error(err, "Failed to parse polling interval, using default", "interval", config.Spec.PollingInterval)
		} else {
			requeueAfter = parsed
		}
	}

	// 3. Build pricing config from spec.
	pricingConfig := pricingFromSpec(config.Spec.Pricing)

	// 4. Create collector.
	// NOTE: Once #2 (metrics-server collector) is merged, update to pass pricingConfig.
	_ = pricingConfig // Will be used by the collector after #2 is merged.
	coll := collector.NewMetricsServerCollector("")

	// 5. Get or create detector for this KostConfig.
	det := r.getOrCreateDetector(req.String(), config.Spec)

	// 6. Build explainer from spec.
	exp := explainerFromSpec(config.Spec.Explainer)

	// 7. Collect metrics.
	usages, err := coll.Collect(ctx)
	if err != nil {
		log.Error(err, "Failed to collect metrics")
		meta.SetStatusCondition(&config.Status.Conditions, metav1.Condition{
			Type:               conditionTypeDegraded,
			Status:             metav1.ConditionTrue,
			Reason:             "CollectionFailed",
			Message:            fmt.Sprintf("Failed to collect metrics: %v", err),
			LastTransitionTime: metav1.Now(),
		})
		if updateErr := r.Status().Update(ctx, &config); updateErr != nil {
			log.Error(updateErr, "Failed to update status after collection error")
		}
		return ctrl.Result{RequeueAfter: requeueAfter}, nil
	}

	// 8. Update baselines with current data.
	det.UpdateBaselines(usages)

	// 9. Detect anomalies.
	anomalies := det.Detect(usages)
	log.Info("Completed anomaly detection", "namespaces", len(usages), "anomalies", len(anomalies))

	// 10. Explain and alert for each anomaly.
	for _, anomaly := range anomalies {
		explanation, explainErr := exp.Explain(ctx, anomaly)
		if explainErr != nil {
			log.Error(explainErr, "Failed to explain anomaly", "namespace", anomaly.Namespace)
			continue
		}
		log.Info("Anomaly detected",
			"namespace", anomaly.Namespace,
			"currentCost", anomaly.CurrentCost,
			"baselineCost", anomaly.BaselineCost,
			"deviation", fmt.Sprintf("%.0f%%", anomaly.DeviationPercent),
			"summary", explanation.Summary,
		)
	}

	// 11. Update status.
	now := metav1.Now()
	config.Status.LastCheckTime = &now
	config.Status.ActiveAnomalies = len(anomalies)

	meta.SetStatusCondition(&config.Status.Conditions, metav1.Condition{
		Type:               conditionTypeAvailable,
		Status:             metav1.ConditionTrue,
		Reason:             "ReconcileSucceeded",
		Message:            fmt.Sprintf("Checked %d namespaces, found %d anomalies", len(usages), len(anomalies)),
		LastTransitionTime: now,
	})
	meta.RemoveStatusCondition(&config.Status.Conditions, conditionTypeDegraded)

	if err := r.Status().Update(ctx, &config); err != nil {
		log.Error(err, "Failed to update status")
		return ctrl.Result{}, err
	}

	return ctrl.Result{RequeueAfter: requeueAfter}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *KostConfigReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&kostv1alpha1.KostConfig{}).
		Named("kostconfig").
		Complete(r)
}

// getOrCreateDetector returns an existing detector for the given key or creates
// a new one from the KostConfig spec.
func (r *KostConfigReconciler) getOrCreateDetector(key string, spec kostv1alpha1.KostConfigSpec) *detector.Detector {
	r.detectorsMu.Lock()
	defer r.detectorsMu.Unlock()

	if det, ok := r.detectors[key]; ok {
		return det
	}

	cfg := detector.DetectorConfig{
		SigmaThreshold: spec.SigmaThreshold,
		MinSamples:     spec.MinSamples,
	}
	if cfg.SigmaThreshold == 0 {
		cfg.SigmaThreshold = detector.DefaultDetectorConfig().SigmaThreshold
	}
	if cfg.MinSamples == 0 {
		cfg.MinSamples = detector.DefaultDetectorConfig().MinSamples
	}

	det := detector.NewDetector(cfg)
	r.detectors[key] = det
	return det
}

// pricingFromSpec converts a PricingSpec to a pricing.PricingConfig,
// falling back to defaults for zero values.
func pricingFromSpec(spec kostv1alpha1.PricingSpec) pricing.PricingConfig {
	cfg := pricing.DefaultPricing()
	if spec.CPUPerCoreHour > 0 {
		cfg.CPUPerCoreHour = spec.CPUPerCoreHour
	}
	if spec.MemoryPerGBHour > 0 {
		cfg.MemoryPerGBHour = spec.MemoryPerGBHour
	}
	if spec.GPUPerHour > 0 {
		cfg.GPUPerHour = spec.GPUPerHour
	}
	return cfg
}

// explainerFromSpec creates an Explainer based on the ExplainerSpec provider field.
func explainerFromSpec(spec kostv1alpha1.ExplainerSpec) explainer.Explainer {
	// For now, only noop is implemented. Other providers will be added
	// as their respective issues are completed.
	switch spec.Provider {
	case "claude", "openai", "ollama":
		// TODO: Implement when respective explainer issues are done.
		// Fall through to noop for now.
		return explainer.NewNoopExplainer()
	default:
		return explainer.NewNoopExplainer()
	}
}
