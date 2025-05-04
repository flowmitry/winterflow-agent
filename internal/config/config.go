package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const (
	// DefaultGRPCServerAddress is the default gRPC server address for agent communication
	DefaultGRPCServerAddress = "localhost:8081"
	// DefaultAPIBaseURL is the default HTTP API server URL for web interface
	DefaultAPIBaseURL = "http://localhost:8080"
	// DefaultAnsiblePath is the default path for Ansible files
	DefaultAnsiblePath = "ansible"
	// DefaultAppsPath is the default path for application files
	DefaultAppsPath = "apps"
	// DefaultAnsibleAppsPath is the default path for Ansible application files
	DefaultAnsibleAppsPath = "ansible_apps"
)

// Config holds the application configuration
type Config struct {
	ServerID    string          `json:"server_id"`
	ServerToken string          `json:"server_token"`
	Features    map[string]bool `json:"features"`
	// GRPCServerAddress is the gRPC server address for agent communication
	GRPCServerAddress string `json:"grpc_server_address,omitempty"`
	// APIBaseURL is the base HTTP API URL for web interface
	APIBaseURL string `json:"api_base_url,omitempty"`
	// AnsiblePath is the path where ansible files are stored
	AnsiblePath string `json:"ansible_path,omitempty"`
	// AppsPath is the path where application files are stored
	AppsPath string `json:"apps_path,omitempty"`
	// AnsibleAppsPath is the path where ansible application files are stored
	AnsibleAppsPath string `json:"ansible_apps_path,omitempty"`
}

// applyDefaults ensures that all necessary fields have default values if they are empty.
func applyDefaults(cfg *Config) {
	if cfg.GRPCServerAddress == "" {
		cfg.GRPCServerAddress = DefaultGRPCServerAddress
	}
	if cfg.APIBaseURL == "" {
		cfg.APIBaseURL = DefaultAPIBaseURL
	}
	if cfg.AnsiblePath == "" {
		cfg.AnsiblePath = DefaultAnsiblePath
	}
	if cfg.AppsPath == "" {
		cfg.AppsPath = DefaultAppsPath
	}
	if cfg.AnsibleAppsPath == "" {
		cfg.AnsibleAppsPath = DefaultAnsibleAppsPath
	}
}

// validateAndMergeFeatures ensures only supported features are used and merges with defaults
func validateAndMergeFeatures(configFeatures map[string]bool) map[string]bool {
	if configFeatures == nil {
		configFeatures = make(map[string]bool)
	}

	// Create a new map with default values
	mergedFeatures := make(map[string]bool)
	for feature, defaultValue := range DefaultFeatureValues {
		// If the feature is defined in config, use that value
		if value, exists := configFeatures[feature]; exists {
			mergedFeatures[feature] = value
		} else {
			// Otherwise use the default value
			mergedFeatures[feature] = defaultValue
		}
	}

	return mergedFeatures
}

// LoadConfig loads the configuration from a JSON file
func LoadConfig(configPath string) (*Config, error) {
	// Create a new config struct (defaults will be applied later)
	config := &Config{
		Features: make(map[string]bool),
	}

	// Set default features initially
	config.Features = validateAndMergeFeatures(nil)

	// Try to load existing config if it exists
	if _, err := os.Stat(configPath); err == nil {
		data, err := os.ReadFile(configPath)
		if err == nil {
			if err := json.Unmarshal(data, config); err == nil {
				// Validate and merge features from the loaded config
				config.Features = validateAndMergeFeatures(config.Features)
				// Apply defaults to the loaded config (overwriting empty fields)
				applyDefaults(config)
				// Config loaded and defaults applied, return it
				return config, nil
			}
		}
	}

	// If file doesn't exist or any error occurred during loading,
	// apply defaults to the initial empty config structure.
	applyDefaults(config)
	return config, nil
}

// WaitUntilCompleted waits for the configuration file to exist and have valid content
func WaitUntilReady(configPath string) (*Config, error) {
	for {
		if _, err := os.Stat(configPath); err == nil {
			// Try to read and validate the config
			data, err := os.ReadFile(configPath)
			if err == nil {
				var config Config // Start with an empty config
				if err := json.Unmarshal(data, &config); err == nil {
					// Check if required fields are filled
					if config.ServerID != "" && config.ServerToken != "" {
						// All required fields are present, proceed
						// Validate and merge features
						config.Features = validateAndMergeFeatures(config.Features)
						// Apply defaults for optional fields
						applyDefaults(&config)
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

	// Ensure default values are set before saving
	applyDefaults(config)

	// Validate and merge features before saving
	config.Features = validateAndMergeFeatures(config.Features)

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
