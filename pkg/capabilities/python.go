package capabilities

import (
	"os/exec"
	"strings"
)

// PythonCapability represents the Python capability
type PythonCapability struct {
	version string
}

// NewPythonCapability creates a new Python capability
// Returns nil if Python is not available
func NewPythonCapability() *PythonCapability {
	capability := &PythonCapability{
		version: "",
	}

	// Check if Python is available
	cmd := exec.Command("python3", "--version")
	output, err := cmd.Output()
	if err != nil {
		return capability
	}

	// Parse version from output
	versionStr := string(output)
	if strings.Contains(versionStr, "Python") {
		// Extract version from output
		parts := strings.Split(versionStr, " ")
		if len(parts) > 1 {
			capability.version = strings.TrimSpace(parts[1])
		}
		return capability
	}
	return capability
}

// Name returns the name of the capability
func (c *PythonCapability) Name() string {
	return CapabilityPython
}

// Value returns the value of the capability
func (c *PythonCapability) Value() string {
	return c.version
}
