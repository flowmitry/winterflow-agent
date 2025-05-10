package capabilities

// Capability names
const (
	CapabilityAnsible       = "ansible"
	CapabilityPython        = "python"
	CapabilityDocker        = "docker"
	CapabilityDockerCompose = "docker-compose"
	CapabilityDockerSwarm   = "docker-swarm"
	// System info capabilities
	SystemCapabilityCpuCores    = "system_cpu_cores"
	CapabilitySystemUptime      = "system_uptime"
	CapabilitySystemMemoryTotal = "system_memory_total"
	CapabilitySystemDiskTotal   = "system_disk_total"
	// OS capabilities
	CapabilityOS     = "os"
	CapabilityOSArch = "os_arch"
)

// Capability represents a system capability that can be detected
type Capability interface {
	// Name returns the name of the capability
	Name() string
	// Value returns the value of the capability
	Value() string
	// IsAvailable returns whether the capability is available
	IsAvailable() bool
}

// SystemInfo represents basic system information
type SystemInfo struct {
	OS   string
	Arch string
}

// CapabilityFactory creates and returns all available capabilities
type CapabilityFactory struct {
	capabilities []Capability
}

// NewCapabilityFactory creates a new capability factory
func NewCapabilityFactory() *CapabilityFactory {
	return &CapabilityFactory{
		capabilities: []Capability{
			NewAnsibleCapability(),
			NewPythonCapability(),
			NewDockerCapability(),
			NewDockerComposeCapability(),
			NewDockerSwarmCapability(),
			// System info capabilities
			NewSystemCpuCoresCapability(),
			NewSystemUptimeCapability(),
			NewSystemMemoryTotalCapability(),
			NewSystemDiskTotalCapability("/"),
			// OS capabilities
			NewSystemOSCapability(),
			NewSystemOSArchCapability(),
		},
	}
}

// GetAllCapabilities returns all available capabilities
func (f *CapabilityFactory) GetAllCapabilities() []Capability {
	return f.capabilities
}

// GetCapabilityByName returns a capability by its name
func (f *CapabilityFactory) GetCapabilityByName(name string) Capability {
	for _, capability := range f.capabilities {
		if capability.Name() == name {
			return capability
		}
	}
	return nil
}
