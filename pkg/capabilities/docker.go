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
func NewDockerCapability() *DockerCapability {
	return &DockerCapability{
		version: "20.10", // Default version
	}
}

// Name returns the name of the capability
func (c *DockerCapability) Name() string {
	return CapabilityDocker
}

// Version returns the version of the capability
func (c *DockerCapability) Version() string {
	return c.version
}

// IsAvailable checks if Docker is available on the system
func (c *DockerCapability) IsAvailable() bool {
	cmd := exec.Command("docker", "--version")
	output, err := cmd.Output()
	if err != nil {
		return false
	}

	// Parse version from output
	versionStr := string(output)
	if strings.Contains(versionStr, "Docker version") {
		// Extract version from output
		parts := strings.Split(versionStr, " ")
		if len(parts) > 2 {
			c.version = parts[2]
		}
		return true
	}
	return false
}
