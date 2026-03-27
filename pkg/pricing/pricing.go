// Package pricing provides configurable resource cost estimation
// for Kubernetes workloads.
package pricing

// PricingConfig holds per-resource hourly prices used to estimate
// the cost of running workloads.
type PricingConfig struct {
	// CPUPerCoreHour is the cost per CPU core per hour.
	CPUPerCoreHour float64 `json:"cpuPerCoreHour"`

	// MemoryPerGBHour is the cost per GB of memory per hour.
	MemoryPerGBHour float64 `json:"memoryPerGBHour"`

	// GPUPerHour is the cost per GPU per hour.
	GPUPerHour float64 `json:"gpuPerHour"`
}

// DefaultPricing returns a PricingConfig with approximate AWS on-demand
// prices as sensible defaults.
func DefaultPricing() PricingConfig {
	return PricingConfig{
		CPUPerCoreHour:  0.0340, // ~m5.xlarge equivalent
		MemoryPerGBHour: 0.0043, // ~m5.xlarge equivalent
		GPUPerHour:      0.5260, // ~g4dn.xlarge equivalent
	}
}

// EstimateCost calculates the estimated hourly cost for the given resources.
// CPU is in cores (not millicores), memory is in GB, gpu is count.
func (p PricingConfig) EstimateCost(cpuCores, memoryGB float64, gpuCount int) float64 {
	return (cpuCores * p.CPUPerCoreHour) +
		(memoryGB * p.MemoryPerGBHour) +
		(float64(gpuCount) * p.GPUPerHour)
}

// EstimateCostFromMillis is a convenience method that accepts CPU in millicores
// and memory in bytes, converting them before estimation.
func (p PricingConfig) EstimateCostFromMillis(cpuMillicores, memoryBytes int64, gpuCount int) float64 {
	cpuCores := float64(cpuMillicores) / 1000.0
	memoryGB := float64(memoryBytes) / (1024 * 1024 * 1024)
	return p.EstimateCost(cpuCores, memoryGB, gpuCount)
}
