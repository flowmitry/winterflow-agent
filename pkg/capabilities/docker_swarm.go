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
func NewDockerSwarmCapability() *DockerSwarmCapability {
	return &DockerSwarmCapability{
		version: "2.1", // Default version
	}
}

// Name returns the name of the capability
func (c *DockerSwarmCapability) Name() string {
	return CapabilityDockerSwarm
}

// Version returns the version of the capability
func (c *DockerSwarmCapability) Version() string {
	return c.version
}

// IsAvailable checks if Docker Swarm is available on the system
func (c *DockerSwarmCapability) IsAvailable() bool {
	// Docker Swarm is part of Docker, so we check if Docker is available first
	dockerCmd := exec.Command("docker", "info")
	_, err := dockerCmd.Output()
	if err != nil {
		return false
	}

	// Check if Swarm is initialized
	swarmCmd := exec.Command("docker", "node", "ls")
	swarmOutput, err := swarmCmd.Output()
	if err != nil {
		return false
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
					c.version = parts[2]
				}
			}
		}
		return true
	}
	return false
}
