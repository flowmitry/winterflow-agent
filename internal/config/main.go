package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

const (
	DefaultPort = 18080 // Default port for the application to listen on
)

// Config represents the agent configuration
type Config struct {
	WorkingDir    string `json:"working_dir"`    // Directory where the agent stores its files
	PlaybooksPath string `json:"playbooks_path"` // Path to the winterflow playbooks
	DeviceID      string `json:"device_id"`
	AgentToken    string `json:"agent_token"`   // Token received after registration
	RegisteredAt  string `json:"registered_at"` // ISO 8601 timestamp
	Port          int    `json:"port"`          // Port for the application to listen on
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

	return &config, nil
}

// SaveConfig saves the configuration to the specified file
func (m *Manager) SaveConfig(config *Config) error {
	// Validate that device ID is not empty
	if config.DeviceID == "" {
		return fmt.Errorf("device ID cannot be empty")
	}

	// Set default port if not specified
	if config.Port == 0 {
		config.Port = DefaultPort
	}

	// Set default working directory if not specified
	if config.WorkingDir == "" {
		exe, err := os.Executable()
		if err != nil {
			return fmt.Errorf("failed to get executable path: %w", err)
		}
		config.WorkingDir = filepath.Dir(exe)
	}

	// Set default playbooks path if not specified
	if config.PlaybooksPath == "" {
		config.PlaybooksPath = filepath.Join(config.WorkingDir, "ansible-playbooks")
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
