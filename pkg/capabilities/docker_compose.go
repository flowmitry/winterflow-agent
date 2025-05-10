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
// Returns nil if Docker Compose is not available
func NewDockerComposeCapability() *DockerComposeCapability {
	capability := &DockerComposeCapability{
		version: "",
	}

	// Check if Docker Compose is available
	cmd := exec.Command("docker-compose", "--version")
	output, err := cmd.Output()
	if err != nil {
		return capability
	}

	// Parse version from output
	versionStr := string(output)
	if strings.Contains(versionStr, "Docker Compose version") {
		// Extract version from output
		parts := strings.Split(versionStr, " ")
		if len(parts) > 3 {
			capability.version = parts[3]
		}
		return capability
	}
	return capability
}

// Name returns the name of the capability
func (c *DockerComposeCapability) Name() string {
	return CapabilityDockerCompose
}

// Value returns the value of the capability
func (c *DockerComposeCapability) Value() string {
	return c.version
}
