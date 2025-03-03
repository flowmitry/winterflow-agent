// Package winterflow provides the main interface for the Winterflow agent
package winterflow

import (
	"winterflow-agent/internal/config"
	"winterflow-agent/internal/winterflow/client"
	"winterflow-agent/internal/winterflow/registration"
)

// Register starts the device flow registration process
func Register(configPath string) error {
	return registration.Register(configPath)
}

// NewClient creates a new Winterflow API client
func NewClient(config *config.Config) (*client.Client, error) {
	return client.NewClient(config.AgentToken, config.DeviceID)
}
