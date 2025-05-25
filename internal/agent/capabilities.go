package agent

import (
	"winterflow-agent/pkg/capabilities"
)

func GetCapabilities() *CapabilityFactory {
	return NewCapabilityFactory()
}

// CapabilityFactory creates and returns all available capabilities
type CapabilityFactory struct {
	capabilities []capabilities.Capability
}

// NewCapabilityFactory creates a new capability factory
func NewCapabilityFactory() *CapabilityFactory {
	return &CapabilityFactory{
		capabilities: []capabilities.Capability{
			capabilities.NewAnsibleCapability(),
			capabilities.NewPythonCapability(),
			capabilities.NewDockerCapability(),
			capabilities.NewDockerComposeCapability(),
			capabilities.NewDockerSwarmCapability(),
			// System info capabilities
			capabilities.NewSystemCpuCoresCapability(),
			capabilities.NewSystemUptimeCapability(),
			capabilities.NewSystemMemoryTotalCapability(),
			capabilities.NewSystemDiskTotalCapability("/"),
			// OS capabilities
			capabilities.NewSystemOSCapability(),
			capabilities.NewSystemOSArchCapability(),
			// Agent capabilities
			capabilities.NewAgentVersionCapability(),
		},
	}
}

func (f *CapabilityFactory) ToMap() map[string]string {
	result := make(map[string]string)
	for _, capability := range f.capabilities {
		result[capability.Name()] = capability.Value()
	}
	return result
}
