package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
	log "winterflow-agent/pkg/log"
)

// AgentStatus represents the current status of the agent
type AgentStatus string

const (
	// AgentStatusRegistered indicates the agent is registered with the server
	AgentStatusRegistered AgentStatus = "registered"
	// AgentStatusPending indicates the agent registration is pending
	AgentStatusPending AgentStatus = "pending"
	// AgentStatusUnknown indicates the agent status is unknown
	AgentStatusUnknown AgentStatus = "unknown"
)

var (
	// DefaultGRPCServerAddress is the default gRPC server address for agent communication
	DefaultGRPCServerAddress = "grpc.winterflow.io:50051"
	// DefaultAPIBaseURL is the default HTTP API server URL for web interface
	DefaultAPIBaseURL = "https://app.winterflow.io"
)

const (
	// DefaultAnsiblePath is the default path for Ansible files
	DefaultAnsiblePath = "ansible"
	// AnsibleAppsRolesFolder defines the folder name where Ansible application role files are stored.
	AnsibleAppsRolesFolder               = "apps_roles"
	AnsibleAppsRolesCurrentVersionFolder = "current"

	// DefaultLogsPath is the default directory path where application log files are stored.
	DefaultLogsPath = "/var/log/winterflow"

	// DefaultCertificatesPath is the default directory path for storing certificates.
	DefaultCertificatesPath = ".certs"
	// DefaultAgentPrivateKeyPath is the default path for the agent's private key
	DefaultAgentPrivateKeyPath = ".certs/agent.key"
	// DefaultCSRPath is the default path for the Certificate Signing Request
	DefaultCSRPath = ".certs/agent.csr"
	// DefaultCertificatePath is the default path for the signed certificate
	DefaultCertificatePath = ".certs/agent.crt"
	// DefaultCACertificatePath is the default filesystem path for the trusted Certificate Authority (CA) certificate.
	DefaultCACertificatePath = ".certs/ca.crt"
)

// Config holds the application configuration
type Config struct {
	AgentID     string          `json:"agent_id"`
	AgentStatus AgentStatus     `json:"agent_status"`
	Features    map[string]bool `json:"features"`
	// GRPCServerAddress is the gRPC server address for agent communication
	GRPCServerAddress string `json:"grpc_server_address,omitempty"`
	// APIBaseURL is the base HTTP API URL for web interface
	APIBaseURL string `json:"api_base_url,omitempty"`
	// AnsiblePath is the path where ansible files are stored
	AnsiblePath string `json:"ansible_path,omitempty"`
	// LogsPath specifies the directory where log files are stored.
	LogsPath string `json:"logs_path,omitempty"`
	// CertificatesPath is the path where certificate-related files are stored.
	CertificatesPath string `json:"certificates_path,omitempty"`
	// CACertificatePath is the path where the Certificate Authority's certificate is stored.
	CACertificatePath string `json:"ca_certificate_path,omitempty"`
	// PrivateKeyPath is the path where the agent's private key is stored
	PrivateKeyPath string `json:"private_key_path,omitempty"`
	// CSRPath is the path where the Certificate Signing Request is stored
	CSRPath string `json:"csr_path,omitempty"`
	// CertificatePath is the path where the signed certificate is stored
	CertificatePath string `json:"certificate_path,omitempty"`
}

// applyDefaults ensures that all necessary fields have default values if they are empty.
func applyDefaults(cfg *Config) {
	if cfg.AgentStatus == "" {
		cfg.AgentStatus = AgentStatusUnknown
	}
	if cfg.GRPCServerAddress == "" {
		cfg.GRPCServerAddress = DefaultGRPCServerAddress
	}
	if cfg.APIBaseURL == "" {
		cfg.APIBaseURL = DefaultAPIBaseURL
	}
	if cfg.AnsiblePath == "" {
		cfg.AnsiblePath = DefaultAnsiblePath
	}
	if cfg.LogsPath == "" {
		cfg.LogsPath = DefaultLogsPath
	}
	if cfg.CertificatesPath == "" {
		cfg.CertificatesPath = DefaultCertificatesPath
	}
	if cfg.PrivateKeyPath == "" {
		cfg.PrivateKeyPath = DefaultAgentPrivateKeyPath
	}
	if cfg.CSRPath == "" {
		cfg.CSRPath = DefaultCSRPath
	}
	if cfg.CertificatePath == "" {
		cfg.CertificatePath = DefaultCertificatePath
	}
	if cfg.CACertificatePath == "" {
		cfg.CACertificatePath = DefaultCACertificatePath
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
					// Check if required fields are filled and agent is registered
					if config.AgentID != "" && config.AgentStatus == AgentStatusRegistered {
						// All required fields are present and agent is registered, proceed
						// Validate and merge features
						config.Features = validateAndMergeFeatures(config.Features)
						// Apply defaults for optional fields
						applyDefaults(&config)
						return &config, nil
					}
				}
			}
		}
		log.Printf("Waiting for valid configuration file with registered status at %s...", configPath)
		time.Sleep(5 * time.Second)
	}
}

// SaveConfig saves the configuration to a JSON file
func SaveConfig(config *Config, configPath string) error {
	// Create directory if it doesn't exist
	dir := filepath.Dir(configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return log.Errorf("failed to create config directory: %v", err)
	}

	// Ensure default values are set before saving
	applyDefaults(config)

	// Validate and merge features before saving
	config.Features = validateAndMergeFeatures(config.Features)

	// Marshal config to JSON
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return log.Errorf("failed to marshal config: %v", err)
	}

	// Write to file
	if err := os.WriteFile(configPath, data, 0600); err != nil {
		return log.Errorf("failed to write config file: %v", err)
	}

	return nil
}

func (c *Config) GetAnsibleAppsRolesPath() string {
	return fmt.Sprintf("%s/%s", c.AnsiblePath, AnsibleAppsRolesFolder)
}
