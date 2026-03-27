// Package detector implements statistical anomaly detection for
// Kubernetes resource cost data.
package detector

import "time"

// Severity represents the severity level of a detected anomaly.
type Severity string

const (
	SeverityLow      Severity = "low"
	SeverityMedium   Severity = "medium"
	SeverityHigh     Severity = "high"
	SeverityCritical Severity = "critical"
)

// Anomaly represents a detected cost anomaly for a namespace.
type Anomaly struct {
	// Namespace where the anomaly was detected.
	Namespace string `json:"namespace"`

	// CurrentCost is the current estimated hourly cost.
	CurrentCost float64 `json:"currentCost"`

	// BaselineCost is the expected hourly cost based on historical data.
	BaselineCost float64 `json:"baselineCost"`

	// DeviationPercent is how far the current cost deviates from baseline.
	DeviationPercent float64 `json:"deviationPercent"`

	// Severity of the anomaly based on deviation magnitude.
	Severity Severity `json:"severity"`

	// DetectedAt is when the anomaly was detected.
	DetectedAt time.Time `json:"detectedAt"`

	// RecentEvents are relevant Kubernetes events that may explain the anomaly.
	RecentEvents []string `json:"recentEvents,omitempty"`
}

// Baseline represents the historical cost baseline for a namespace.
type Baseline struct {
	// Namespace this baseline belongs to.
	Namespace string `json:"namespace"`

	// AvgCost is the rolling average hourly cost.
	AvgCost float64 `json:"avgCost"`

	// StdDev is the standard deviation of the hourly cost.
	StdDev float64 `json:"stdDev"`

	// SampleCount is the number of samples used to compute the baseline.
	SampleCount int `json:"sampleCount"`

	// LastUpdated is when the baseline was last recalculated.
	LastUpdated time.Time `json:"lastUpdated"`
}

// DetectorConfig holds configuration for the anomaly detector.
type DetectorConfig struct {
	// SigmaThreshold is the number of standard deviations above the mean
	// that triggers an anomaly. Default: 2.0.
	SigmaThreshold float64 `json:"sigmaThreshold"`

	// MinSamples is the minimum number of data points required before
	// anomaly detection activates. Default: 10.
	MinSamples int `json:"minSamples"`
}

// DefaultDetectorConfig returns a DetectorConfig with sensible defaults.
func DefaultDetectorConfig() DetectorConfig {
	return DetectorConfig{
		SigmaThreshold: 2.0,
		MinSamples:     10,
	}
}
