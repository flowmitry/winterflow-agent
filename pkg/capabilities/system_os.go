package capabilities

import (
	"runtime"
)

// SystemOSCapability reports the operating system information.
type SystemOSCapability struct {
	osType string
	arch   string
}

// NewSystemOSCapability returns a new SystemOSCapability.
func NewSystemOSCapability() *SystemOSCapability {
	return &SystemOSCapability{
		osType: runtime.GOOS,
		arch:   runtime.GOARCH,
	}
}

// Name implements Capability.
func (c *SystemOSCapability) Name() string {
	return CapabilityOS
}

// Value implements Capability.
func (c *SystemOSCapability) Value() string {
	return c.osType
}

// GetArch returns the OS architecture.
func (c *SystemOSCapability) GetArch() string {
	return c.arch
}

// SystemOSArchCapability reports the operating system architecture.
type SystemOSArchCapability struct {
	osCapability *SystemOSCapability
}

// NewSystemOSArchCapability returns a new SystemOSArchCapability.
func NewSystemOSArchCapability() *SystemOSArchCapability {
	return &SystemOSArchCapability{
		osCapability: NewSystemOSCapability(),
	}
}

// Name implements Capability.
func (c *SystemOSArchCapability) Name() string {
	return CapabilityOSArch
}

// Value implements Capability.
func (c *SystemOSArchCapability) Value() string {
	return c.osCapability.GetArch()
}
