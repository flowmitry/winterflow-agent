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
func NewPythonCapability() *PythonCapability {
	return &PythonCapability{
		version: "3.10", // Default version
	}
}

// Name returns the name of the capability
func (c *PythonCapability) Name() string {
	return CapabilityPython
}

// Value returns the value of the capability
func (c *PythonCapability) Value() string {
	return c.version
}

// IsAvailable checks if Python is available on the system
func (c *PythonCapability) IsAvailable() bool {
	cmd := exec.Command("python3", "--version")
	output, err := cmd.Output()
	if err != nil {
		return false
	}

	// Parse version from output
	versionStr := string(output)
	if strings.Contains(versionStr, "Python") {
		// Extract version from output
		parts := strings.Split(versionStr, " ")
		if len(parts) > 1 {
			c.version = parts[1]
		}
		return true
	}
	return false
}
