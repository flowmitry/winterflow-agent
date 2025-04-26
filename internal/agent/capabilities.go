package agent

import (
	"winterflow-agent/pkg/capabilities"
)

// SystemCapabilities represents the system capabilities of the agent
type SystemCapabilities struct {
	OS            string
	OSArch        string
	Ansible       string
	Python        string
	Docker        string
	DockerCompose string
	DockerSwarm   string
	Kubernetes    string
	Git           string
}

// GetSystemCapabilities returns the current system capabilities
func GetSystemCapabilities() SystemCapabilities {
	factory := capabilities.NewCapabilityFactory()
	sysInfo := capabilities.GetSystemInfo()

	// Initialize with system info
	result := SystemCapabilities{
		OS:     sysInfo.OS,
		OSArch: sysInfo.Arch,
	}

	// Get all capabilities
	for _, c := range factory.GetAllCapabilities() {
		if c.IsAvailable() {
			switch c.Name() {
			case capabilities.CapabilityAnsible:
				result.Ansible = c.Version()
			case capabilities.CapabilityPython:
				result.Python = c.Version()
			case capabilities.CapabilityDocker:
				result.Docker = c.Version()
			case capabilities.CapabilityDockerCompose:
				result.DockerCompose = c.Version()
			case capabilities.CapabilityDockerSwarm:
				result.DockerSwarm = c.Version()
			case capabilities.CapabilityKubernetes:
				result.Kubernetes = c.Version()
			case capabilities.CapabilityGit:
				result.Git = c.Version()
			}
		}
	}

	return result
}

// ToMap converts SystemCapabilities to a map[string]string
func (c SystemCapabilities) ToMap() map[string]string {
	return map[string]string{
		"os":             c.OS,
		"arch":           c.OSArch,
		"ansible":        c.Ansible,
		"python":         c.Python,
		"docker":         c.Docker,
		"docker-compose": c.DockerCompose,
		"docker-swarm":   c.DockerSwarm,
		"kubernetes":     c.Kubernetes,
		"git":            c.Git,
	}
}
