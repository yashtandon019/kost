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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// KostConfigSpec defines the desired state of KostConfig.
type KostConfigSpec struct {
	// PollingInterval is how often to check for anomalies (e.g., "5m", "30s").
	// +kubebuilder:default="5m"
	// +optional
	PollingInterval string `json:"pollingInterval,omitempty"`

	// SigmaThreshold is the number of standard deviations above the mean
	// that triggers an anomaly alert.
	// +kubebuilder:default=2.0
	// +kubebuilder:validation:Minimum=0.5
	// +optional
	SigmaThreshold float64 `json:"sigmaThreshold,omitempty"`

	// MinSamples is the minimum number of data points required before
	// anomaly detection activates.
	// +kubebuilder:default=10
	// +kubebuilder:validation:Minimum=3
	// +optional
	MinSamples int `json:"minSamples,omitempty"`

	// Explainer configures the LLM backend for generating explanations.
	// +optional
	Explainer ExplainerSpec `json:"explainer,omitempty"`

	// Pricing configures per-resource cost estimation.
	// +optional
	Pricing PricingSpec `json:"pricing,omitempty"`
}

// ExplainerSpec configures the LLM backend used for anomaly explanations.
type ExplainerSpec struct {
	// Provider is the LLM backend to use: "noop", "claude", "openai", or "ollama".
	// +kubebuilder:default="noop"
	// +kubebuilder:validation:Enum=noop;claude;openai;ollama
	// +optional
	Provider string `json:"provider,omitempty"`

	// Model is the specific model to use (e.g., "claude-sonnet-4-20250514", "gpt-4o").
	// +optional
	Model string `json:"model,omitempty"`

	// Endpoint is a custom API endpoint URL (useful for Ollama or proxies).
	// +optional
	Endpoint string `json:"endpoint,omitempty"`

	// SecretRef references a Kubernetes Secret containing the API key.
	// The secret must have a key named "api-key".
	// +optional
	SecretRef *SecretReference `json:"secretRef,omitempty"`
}

// SecretReference is a reference to a Kubernetes Secret in the same namespace.
type SecretReference struct {
	// Name is the name of the Secret.
	// +required
	Name string `json:"name"`
}

// PricingSpec configures per-resource hourly costs for estimation.
type PricingSpec struct {
	// CPUPerCoreHour is the cost per CPU core per hour.
	// +kubebuilder:default=0.034
	// +optional
	CPUPerCoreHour float64 `json:"cpuPerCoreHour,omitempty"`

	// MemoryPerGBHour is the cost per GB of memory per hour.
	// +kubebuilder:default=0.0043
	// +optional
	MemoryPerGBHour float64 `json:"memoryPerGBHour,omitempty"`

	// GPUPerHour is the cost per GPU per hour.
	// +kubebuilder:default=0.526
	// +optional
	GPUPerHour float64 `json:"gpuPerHour,omitempty"`
}

// KostConfigStatus defines the observed state of KostConfig.
type KostConfigStatus struct {
	// LastCheckTime is when the last anomaly check was performed.
	// +optional
	LastCheckTime *metav1.Time `json:"lastCheckTime,omitempty"`

	// ActiveAnomalies is the number of currently detected anomalies.
	// +optional
	ActiveAnomalies int `json:"activeAnomalies"`

	// conditions represent the current state of the KostConfig resource.
	// +listType=map
	// +listMapKey=type
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// KostConfig is the Schema for the kostconfigs API
type KostConfig struct {
	metav1.TypeMeta `json:",inline"`

	// metadata is a standard object metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitzero"`

	// spec defines the desired state of KostConfig
	// +required
	Spec KostConfigSpec `json:"spec"`

	// status defines the observed state of KostConfig
	// +optional
	Status KostConfigStatus `json:"status,omitzero"`
}

// +kubebuilder:object:root=true

// KostConfigList contains a list of KostConfig
type KostConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitzero"`
	Items           []KostConfig `json:"items"`
}

func init() {
	SchemeBuilder.Register(&KostConfig{}, &KostConfigList{})
}
