package capabilities

import (
	"os/exec"
	"strings"
)

// DockerComposeCapability represents the Docker Compose capability
type DockerComposeCapability struct {
	version string
}

// NewDockerComposeCapability creates a new Docker Compose capability
func NewDockerComposeCapability() *DockerComposeCapability {
	return &DockerComposeCapability{
		version: "2.1", // Default version
	}
}

// Name returns the name of the capability
func (c *DockerComposeCapability) Name() string {
	return CapabilityDockerCompose
}

// Version returns the version of the capability
func (c *DockerComposeCapability) Version() string {
	return c.version
}

// IsAvailable checks if Docker Compose is available on the system
func (c *DockerComposeCapability) IsAvailable() bool {
	cmd := exec.Command("docker-compose", "--version")
	output, err := cmd.Output()
	if err != nil {
		return false
	}

	// Parse version from output
	versionStr := string(output)
	if strings.Contains(versionStr, "Docker Compose version") {
		// Extract version from output
		parts := strings.Split(versionStr, " ")
		if len(parts) > 3 {
			c.version = parts[3]
		}
		return true
	}
	return false
}
