package capabilities

import (
	"runtime"
	"winterflow-agent/pkg/metrics"
)

// SystemDiskTotalCapability reports total bytes in the given mount point.
// Implementation uses the metrics package to get disk information.
// On unsupported OSes (e.g., Windows) returns empty value.
type SystemDiskTotalCapability struct {
	metric metrics.Metric
}

// NewSystemDiskTotalCapability returns a new SystemDiskTotalCapability.
// Returns nil if the OS is Windows, as the implementation uses syscall.Statfs which is not available on Windows.
func NewSystemDiskTotalCapability(path string) *SystemDiskTotalCapability {
	if runtime.GOOS == "windows" {
		return nil
	}
	return &SystemDiskTotalCapability{
		metric: metrics.NewSystemDiskTotalMetric(path),
	}
}

// Name implements Capability.
func (c *SystemDiskTotalCapability) Name() string {
	return CapabilitySystemDiskTotal
}

// Value implements Capability: returns total disk space in bytes.
func (c *SystemDiskTotalCapability) Value() string {
	if c == nil || c.metric == nil {
		return ""
	}
	return c.metric.Value()
}
