package capabilities

import (
	"os/exec"
	"strings"
)

// DockerSwarmCapability represents the Docker Swarm capability
type DockerSwarmCapability struct {
	version string
}

// NewDockerSwarmCapability creates a new Docker Swarm capability
// Returns nil if Docker Swarm is not available
func NewDockerSwarmCapability() *DockerSwarmCapability {
	capability := &DockerSwarmCapability{
		version: "",
	}

	// Docker Swarm is part of Docker, so we check if Docker is available first
	dockerCmd := exec.Command("docker", "info")
	_, err := dockerCmd.Output()
	if err != nil {
		return capability
	}

	// Check if Swarm is initialized
	swarmCmd := exec.Command("docker", "node", "ls")
	swarmOutput, err := swarmCmd.Output()
	if err != nil {
		return capability
	}

	// If we can list nodes, Swarm is available
	if strings.Contains(string(swarmOutput), "ID") {
		// Get Docker version for Swarm version
		versionCmd := exec.Command("docker", "--version")
		versionOutput, err := versionCmd.Output()
		if err == nil {
			versionStr := string(versionOutput)
			if strings.Contains(versionStr, "Docker version") {
				parts := strings.Split(versionStr, " ")
				if len(parts) > 2 {
					capability.version = parts[2]
				}
			}
		}
		return capability
	}
	return capability
}

// Name returns the name of the capability
func (c *DockerSwarmCapability) Name() string {
	return CapabilityDockerSwarm
}

// Value returns the value of the capability
func (c *DockerSwarmCapability) Value() string {
	return c.version
}
