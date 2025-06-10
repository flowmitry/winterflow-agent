package capabilities

import (
	"runtime"
	"winterflow-agent/pkg/metrics"
)

// SystemMemoryTotalCapability reports the total physical memory on Linux.
// Returns empty value on unsupported OSes.
type SystemMemoryTotalCapability struct {
	metric metrics.Metric
}

// NewSystemMemoryTotalCapability returns a new SystemMemoryTotalCapability.
// Returns nil if the OS is not Linux, as the implementation reads from /proc/meminfo which is only available on Linux.
func NewSystemMemoryTotalCapability() *SystemMemoryTotalCapability {
	if runtime.GOOS != "linux" {
		return nil
	}
	return &SystemMemoryTotalCapability{
		metric: metrics.NewSystemMemoryTotalMetric(),
	}
}

// Name implements Capability.
func (c *SystemMemoryTotalCapability) Name() string {
	return CapabilitySystemMemoryTotal
}

// Value implements Capability: reads total memory from metrics.
func (c *SystemMemoryTotalCapability) Value() string {
	if c == nil || c.metric == nil {
		return ""
	}
	return c.metric.Value()
}
