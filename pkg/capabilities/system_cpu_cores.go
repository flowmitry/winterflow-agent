package capabilities

import (
	"runtime"
	"strconv"
)

// SystemCpuCoresCapability represents the number of logical CPU cores available
type SystemCpuCoresCapability struct{}

// NewSystemCpuCoresCapability creates a new CPU cores capability
func NewSystemCpuCoresCapability() *SystemCpuCoresCapability {
	return &SystemCpuCoresCapability{}
}

// Name returns the name of the capability
func (c *SystemCpuCoresCapability) Name() string {
	return SystemCapabilityCpuCores
}

// Value returns the number of CPU cores as a string
func (c *SystemCpuCoresCapability) Value() string {
	return strconv.Itoa(runtime.NumCPU())
}

// IsAvailable always returns true since CPU cores are always available
func (c *SystemCpuCoresCapability) IsAvailable() bool {
	return true
}
