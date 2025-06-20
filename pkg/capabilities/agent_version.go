package capabilities

import (
	"winterflow-agent/internal/application/version"
)

// AgentVersionCapability represents the Agent Version capability
type AgentVersionCapability struct {
	version string
}

// NewAgentVersionCapability creates a new Agent Version capability
func NewAgentVersionCapability() *AgentVersionCapability {
	return &AgentVersionCapability{
		version: version.GetVersion(),
	}
}

// Name returns the name of the capability
func (c *AgentVersionCapability) Name() string {
	return CapabilityAgentVersion
}

// Value returns the value of the capability
func (c *AgentVersionCapability) Value() string {
	return c.version
}
