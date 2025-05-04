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
	// Create a new config with default values
	config := &Config{
		Features:          make(map[string]bool),
		GRPCServerAddress: DefaultGRPCServerAddress,
		APIBaseURL:        DefaultAPIBaseURL,
	}

	// Set default features
	config.Features = validateAndMergeFeatures(nil)

	// Try to load existing config if it exists
	if _, err := os.Stat(configPath); err == nil {
		data, err := os.ReadFile(configPath)
		if err == nil {
			if err := json.Unmarshal(data, config); err == nil {
				// Validate and merge features
				config.Features = validateAndMergeFeatures(config.Features)
				// If we have an API base URL in the config, use it
				if config.APIBaseURL != "" {
					return config, nil
				}
			}
		}
	}

	return config, nil
}

// WaitUntilCompleted waits for the configuration file to exist and have valid content
func WaitUntilReady(configPath string) (*Config, error) {
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
						// Validate and merge features
						config.Features = validateAndMergeFeatures(config.Features)
						if config.GRPCServerAddress == "" {
							config.GRPCServerAddress = DefaultGRPCServerAddress
						}
						if config.APIBaseURL == "" {
							config.APIBaseURL = DefaultAPIBaseURL
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

	// Set default values if not provided
	if config.APIBaseURL == "" {
		config.APIBaseURL = DefaultAPIBaseURL
	}

	// Validate and merge features
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
