package capabilities

import (
	"os/exec"
	"strings"
)

// DockerCapability represents the Docker capability
type DockerCapability struct {
	version string
}

// NewDockerCapability creates a new Docker capability
// Returns nil if Docker is not available
func NewDockerCapability() *DockerCapability {
	capability := &DockerCapability{
		version: "",
	}

	// Check if Docker is available
	cmd := exec.Command("docker", "--version")
	output, err := cmd.Output()
	if err != nil {
		return capability
	}

	// Parse version from output
	versionStr := string(output)
	if strings.Contains(versionStr, "Docker version") {
		// Extract version from output
		parts := strings.Split(versionStr, " ")
		if len(parts) > 2 {
			// Remove trailing comma if present
			capability.version = strings.TrimSuffix(parts[2], ",")
		}
		return capability
	}
	return capability
}

// Name returns the name of the capability
func (c *DockerCapability) Name() string {
	return CapabilityDocker
}

// Value returns the value of the capability
func (c *DockerCapability) Value() string {
	return c.version
}
