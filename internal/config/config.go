package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"winterflow-agent/internal/device"
)

const (
	DefaultPort = 18080 // Default port for the application to listen on
)

// Config represents the agent configuration
type Config struct {
	DeviceID     string `json:"device_id"`
	AgentToken   string `json:"agent_token"`   // Token received after registration
	RegisteredAt string `json:"registered_at"` // ISO 8601 timestamp
	Port         int    `json:"port"`          // Port for the application to listen on
}

// Manager handles configuration loading and saving
type Manager struct {
	configPath string
}

// NewManager creates a new configuration manager
func NewManager(configPath string) *Manager {
	return &Manager{
		configPath: configPath,
	}
}

// LoadConfig loads and validates the configuration from the specified file
func (m *Manager) LoadConfig() (*Config, error) {
	// Check if config file exists
	if _, err := os.Stat(m.configPath); os.IsNotExist(err) {
		return &Config{Port: DefaultPort}, nil // Set default port for new configurations
	}

	data, err := os.ReadFile(m.configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Set default port if not specified
	if config.Port == 0 {
		config.Port = DefaultPort
	}

	// If the config has a device ID, validate it
	if config.DeviceID != "" {
		currentDeviceID, err := device.GetDeviceID()
		if err != nil {
			return nil, fmt.Errorf("failed to get current device ID: %w", err)
		}

		if config.DeviceID != currentDeviceID {
			return nil, fmt.Errorf("configuration device ID mismatch: config has %s but current device is %s",
				config.DeviceID, currentDeviceID)
		}
	}

	return &config, nil
}

// SaveConfig saves the configuration to the specified file
func (m *Manager) SaveConfig(config *Config) error {
	// Validate device ID before saving
	if config.DeviceID != "" {
		currentDeviceID, err := device.GetDeviceID()
		if err != nil {
			return fmt.Errorf("failed to get current device ID: %w", err)
		}

		if config.DeviceID != currentDeviceID {
			return fmt.Errorf("cannot save configuration: device ID mismatch")
		}
	}

	// Create config directory if it doesn't exist
	if err := os.MkdirAll(filepath.Dir(m.configPath), 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(m.configPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// IsRegistered checks if the agent is already registered
func (m *Manager) IsRegistered() (bool, error) {
	cfg, err := m.LoadConfig()
	if err != nil {
		return false, err
	}
	return cfg.AgentToken != "", nil
}
