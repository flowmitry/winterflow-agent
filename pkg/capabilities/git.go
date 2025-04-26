package capabilities

import (
	"os/exec"
	"strings"
)

// GitCapability represents the Git capability
type GitCapability struct {
	version string
}

// NewGitCapability creates a new Git capability
func NewGitCapability() *GitCapability {
	return &GitCapability{
		version: "2.35", // Default version
	}
}

// Name returns the name of the capability
func (c *GitCapability) Name() string {
	return CapabilityGit
}

// Version returns the version of the capability
func (c *GitCapability) Version() string {
	return c.version
}

// IsAvailable checks if Git is available on the system
func (c *GitCapability) IsAvailable() bool {
	cmd := exec.Command("git", "--version")
	output, err := cmd.Output()
	if err != nil {
		return false
	}

	// Parse version from output
	versionStr := string(output)
	if strings.Contains(versionStr, "git version") {
		// Extract version from output
		parts := strings.Split(versionStr, " ")
		if len(parts) > 2 {
			c.version = parts[2]
		}
		return true
	}
	return false
}
