package capabilities

import (
	"os/exec"
	"strings"
)

// KubernetesCapability represents the Kubernetes capability
type KubernetesCapability struct {
	version string
}

// NewKubernetesCapability creates a new Kubernetes capability
func NewKubernetesCapability() *KubernetesCapability {
	return &KubernetesCapability{
		version: "1.25", // Default version
	}
}

// Name returns the name of the capability
func (c *KubernetesCapability) Name() string {
	return CapabilityKubernetes
}

// Version returns the version of the capability
func (c *KubernetesCapability) Version() string {
	return c.version
}

// IsAvailable checks if Kubernetes is available on the system
func (c *KubernetesCapability) IsAvailable() bool {
	// Check if kubectl is available
	cmd := exec.Command("kubectl", "version", "--client")
	output, err := cmd.Output()
	if err != nil {
		return false
	}

	// Parse version from output
	versionStr := string(output)
	if strings.Contains(versionStr, "Client Version") {
		// Extract version from output
		lines := strings.Split(versionStr, "\n")
		for _, line := range lines {
			if strings.Contains(line, "GitVersion") {
				parts := strings.Split(line, "\"")
				if len(parts) > 1 {
					version := parts[1]
					if strings.HasPrefix(version, "v") {
						c.version = version[1:]
					} else {
						c.version = version
					}
				}
				break
			}
		}
		return true
	}
	return false
}
