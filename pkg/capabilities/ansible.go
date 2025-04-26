package capabilities

import (
	"os/exec"
	"strings"
)

// AnsibleCapability represents the Ansible capability
type AnsibleCapability struct {
	version string
}

// NewAnsibleCapability creates a new Ansible capability
func NewAnsibleCapability() *AnsibleCapability {
	return &AnsibleCapability{
		version: "2.12", // Default version
	}
}

// Name returns the name of the capability
func (c *AnsibleCapability) Name() string {
	return CapabilityAnsible
}

// Version returns the version of the capability
func (c *AnsibleCapability) Version() string {
	return c.version
}

// IsAvailable checks if Ansible is available on the system
func (c *AnsibleCapability) IsAvailable() bool {
	cmd := exec.Command("ansible", "--version")
	output, err := cmd.Output()
	if err != nil {
		return false
	}

	// Parse version from output
	versionStr := string(output)
	if strings.Contains(versionStr, "ansible") {
		// Extract version from output
		lines := strings.Split(versionStr, "\n")
		if len(lines) > 0 {
			parts := strings.Split(lines[0], " ")
			if len(parts) > 1 {
				c.version = parts[1]
			}
		}
		return true
	}
	return false
}
