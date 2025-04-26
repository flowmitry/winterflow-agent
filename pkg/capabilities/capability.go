package capabilities

import (
	"runtime"
)

// Capability names
const (
	CapabilityAnsible       = "ansible"
	CapabilityPython        = "python"
	CapabilityDocker        = "docker"
	CapabilityDockerCompose = "docker-compose"
	CapabilityDockerSwarm   = "docker-swarm"
	CapabilityKubernetes    = "kubernetes"
	CapabilityGit           = "git"
)

// Capability represents a system capability that can be detected
type Capability interface {
	// Name returns the name of the capability
	Name() string
	// Version returns the version of the capability
	Version() string
	// IsAvailable returns whether the capability is available
	IsAvailable() bool
}

// SystemInfo represents basic system information
type SystemInfo struct {
	OS   string
	Arch string
}

// GetSystemInfo returns the current system information
func GetSystemInfo() SystemInfo {
	return SystemInfo{
		OS:   runtime.GOOS,
		Arch: runtime.GOARCH,
	}
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
			NewKubernetesCapability(),
			NewGitCapability(),
		},
	}
}

// GetAllCapabilities returns all available capabilities
func (f *CapabilityFactory) GetAllCapabilities() []Capability {
	return f.capabilities
}

// GetCapabilityByName returns a capability by its name
func (f *CapabilityFactory) GetCapabilityByName(name string) Capability {
	for _, cap := range f.capabilities {
		if cap.Name() == name {
			return cap
		}
	}
	return nil
}
