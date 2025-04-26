package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const (
	// DefaultServerAddress is the default gRPC server address
	DefaultServerAddress = "localhost:8081"
)

// Config holds the application configuration
type Config struct {
	ServerID      string          `json:"server_id"`
	ServerToken   string          `json:"server_token"`
	Capabilities  map[string]bool `json:"capabilities"`
	ServerAddress string          `json:"server_address,omitempty"`
}

// LoadConfig loads the configuration from a JSON file
func LoadConfig(configPath string) (*Config, error) {
	// Wait for config file to exist and have valid content
	for {
		if _, err := os.Stat(configPath); err == nil {
			// Try to read and validate the config
			data, err := os.ReadFile(configPath)
			if err == nil {
				var config Config
				if err := json.Unmarshal(data, &config); err == nil {
					// Check if required fields are filled
					if config.ServerID != "" && config.ServerToken != "" {
						// All required fields are present, proceed
						if config.Capabilities == nil {
							config.Capabilities = make(map[string]bool)
						}
						if config.ServerAddress == "" {
							config.ServerAddress = DefaultServerAddress
						}
						return &config, nil
					}
				}
			}
		}
		fmt.Printf("Waiting for valid configuration file at %s...\n", configPath)
		time.Sleep(5 * time.Second)
	}
}

// SaveConfig saves the configuration to a JSON file
func SaveConfig(config *Config, configPath string) error {
	// Create directory if it doesn't exist
	dir := filepath.Dir(configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %v", err)
	}

	// Marshal config to JSON
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %v", err)
	}

	// Write to file
	if err := os.WriteFile(configPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write config file: %v", err)
	}

	return nil
}
