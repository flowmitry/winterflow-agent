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
// Returns nil if Ansible is not available
func NewAnsibleCapability() *AnsibleCapability {
	capability := &AnsibleCapability{
		version: "",
	}

	// Check if Ansible is available
	cmd := exec.Command("ansible", "--version")
	output, err := cmd.Output()
	if err != nil {
		return capability
	}

	// Parse version from output
	versionStr := string(output)
	if strings.Contains(versionStr, "ansible") {
		// Extract version from output
		lines := strings.Split(versionStr, "\n")
		if len(lines) > 0 {
			parts := strings.Split(lines[0], " ")
			if len(parts) > 1 {
				// Handle new format: "ansible [core 2.13.1]"
				if strings.Contains(parts[1], "[core") && len(parts) > 2 {
					capability.version = parts[2]
					// Remove trailing "]" if present
					capability.version = strings.TrimSuffix(capability.version, "]")
				} else {
					capability.version = parts[1]
				}
			}
		}
		return capability
	}
	return capability
}

// Name returns the name of the capability
func (c *AnsibleCapability) Name() string {
	return CapabilityAnsible
}

// Value returns the value of the capability
func (c *AnsibleCapability) Value() string {
	return c.version
}
