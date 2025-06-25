package capabilities

// Capability names
const (
	CapabilityDocker        = "docker"
	CapabilityDockerCompose = "docker_compose"
	CapabilityDockerSwarm   = "docker_swarm"
	// System info capabilities
	SystemCapabilityCpuCores    = "system_cpu_cores"
	CapabilitySystemMemoryTotal = "system_memory_total"
	CapabilitySystemDiskTotal   = "system_disk_total"
	// OS capabilities
	CapabilityOS     = "os"
	CapabilityOSArch = "os_arch"
	// Agent capabilities
	CapabilityAgentVersion = "agent_version"
	CapabilityServerIP     = "server_ip"
)

// Capability represents a system capability that can be detected
type Capability interface {
	// Name returns the name of the capability
	Name() string
	// Value returns the value of the capability
	Value() string
}
