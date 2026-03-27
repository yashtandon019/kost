package detector

import (
	"math"
	"sync"
	"time"

	"github.com/yashtandon019/kost/pkg/collector"
)

// Detector performs statistical anomaly detection on namespace cost data.
type Detector struct {
	mu        sync.RWMutex
	baselines map[string]*Baseline
	config    DetectorConfig
}

// NewDetector creates a new anomaly detector with the given configuration.
func NewDetector(config DetectorConfig) *Detector {
	return &Detector{
		baselines: make(map[string]*Baseline),
		config:    config,
	}
}

// Detect checks current namespace usage against baselines and returns
// any detected anomalies.
func (d *Detector) Detect(usages []collector.NamespaceUsage) []Anomaly {
	d.mu.RLock()
	defer d.mu.RUnlock()

	var anomalies []Anomaly

	for _, usage := range usages {
		baseline, exists := d.baselines[usage.Namespace]
		if !exists || baseline.SampleCount < d.config.MinSamples {
			// Not enough data to detect anomalies yet.
			continue
		}

		if baseline.StdDev == 0 {
			// Avoid division by zero; if stddev is 0, any deviation is anomalous.
			if usage.EstimatedCostPerHour > baseline.AvgCost {
				anomalies = append(anomalies, d.buildAnomaly(usage, baseline))
			}
			continue
		}

		threshold := baseline.AvgCost + (d.config.SigmaThreshold * baseline.StdDev)
		if usage.EstimatedCostPerHour > threshold {
			anomalies = append(anomalies, d.buildAnomaly(usage, baseline))
		}
	}

	return anomalies
}

// UpdateBaselines updates the rolling baselines with new usage data.
func (d *Detector) UpdateBaselines(usages []collector.NamespaceUsage) {
	d.mu.Lock()
	defer d.mu.Unlock()

	for _, usage := range usages {
		baseline, exists := d.baselines[usage.Namespace]
		if !exists {
			d.baselines[usage.Namespace] = &Baseline{
				Namespace:   usage.Namespace,
				AvgCost:     usage.EstimatedCostPerHour,
				StdDev:      0,
				SampleCount: 1,
				LastUpdated: time.Now(),
			}
			continue
		}

		// Welford's online algorithm for computing running mean and variance.
		baseline.SampleCount++
		n := float64(baseline.SampleCount)
		delta := usage.EstimatedCostPerHour - baseline.AvgCost
		baseline.AvgCost += delta / n
		delta2 := usage.EstimatedCostPerHour - baseline.AvgCost
		// Running variance (we store stddev, so convert).
		variance := baseline.StdDev * baseline.StdDev
		variance += (delta*delta2-variance)/n
		baseline.StdDev = math.Sqrt(math.Max(0, variance))
		baseline.LastUpdated = time.Now()
	}
}

// GetBaseline returns the current baseline for a namespace, if it exists.
func (d *Detector) GetBaseline(namespace string) (Baseline, bool) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	b, ok := d.baselines[namespace]
	if !ok {
		return Baseline{}, false
	}
	return *b, true
}

func (d *Detector) buildAnomaly(usage collector.NamespaceUsage, baseline *Baseline) Anomaly {
	deviation := 0.0
	if baseline.AvgCost > 0 {
		deviation = ((usage.EstimatedCostPerHour - baseline.AvgCost) / baseline.AvgCost) * 100
	}

	return Anomaly{
		Namespace:        usage.Namespace,
		CurrentCost:      usage.EstimatedCostPerHour,
		BaselineCost:     baseline.AvgCost,
		DeviationPercent: deviation,
		Severity:         classifySeverity(deviation),
		DetectedAt:       time.Now(),
	}
}

func classifySeverity(deviationPercent float64) Severity {
	switch {
	case deviationPercent >= 500:
		return SeverityCritical
	case deviationPercent >= 200:
		return SeverityHigh
	case deviationPercent >= 100:
		return SeverityMedium
	default:
		return SeverityLow
	}
}
