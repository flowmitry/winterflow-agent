// Package winterflow provides the main interface for the Winterflow agent
package winterflow

import (
	"fmt"
	"winterflow-agent/internal/config"
	"winterflow-agent/internal/winterflow/client"
	"winterflow-agent/internal/winterflow/registration"
)

// Register starts the device flow registration process
func Register(configPath string) error {
	return registration.Register(configPath)
}

// NewClient creates a new Winterflow API client
func NewClient(configPath string) (*client.Client, error) {
	configManager := config.NewManager(configPath)
	cfg, err := configManager.LoadConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}
	return client.NewClient(cfg.AgentToken, cfg.DeviceID)
}
