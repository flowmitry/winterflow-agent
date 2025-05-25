package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
	"winterflow-agent/pkg/log"
)

// AgentStatus represents the current status of the agent
type AgentStatus string

// OrchestratorType represents the orchestrator
type OrchestratorType string

const (
	// AgentStatusRegistered indicates the agent is registered with the server
	AgentStatusRegistered AgentStatus = "registered"
	// AgentStatusPending indicates the agent registration is pending
	AgentStatusPending AgentStatus = "pending"
	// AgentStatusUnknown indicates the agent status is unknown
	AgentStatusUnknown AgentStatus = "unknown"
)

const (
	// OrchestratorTypeDockerCompose represents Docker Compose orchestration
	OrchestratorTypeDockerCompose OrchestratorType = "docker_compose"
	// OrchestratorTypeDockerSwarm represents Docker Swarm orchestration
	OrchestratorTypeDockerSwarm OrchestratorType = "docker_swarm"
	// defaultOrchestrator defines the default container orchestration tool used by the system.
	defaultOrchestrator = OrchestratorTypeDockerCompose
)

var (
	GRPCServerAddress string
	APIBaseURL        string
	BasePath          string
	LogsPath          string
	Orchestrator      OrchestratorType
)

const (
	// defaultGRPCServerAddress is the default gRPC server address for agent communication
	defaultGRPCServerAddress = "grpc.winterflow.io:50051"
	// defaultAPIBaseURL is the default HTTP API server URL for web interface
	defaultAPIBaseURL = "https://app.winterflow.io"
	// defaultBasePath defines the default file system path used by the application for storing and accessing resources.
	defaultBasePath = "/opt/winterflow"
	// defaultLogsPath is the default directory path where application log files are stored.
	defaultLogsPath = "/var/log/winterflow"

	// ansibleFolder is the path for Ansible files
	ansibleFolder = "ansible"
	// ansibleAppsRolesFolder defines the folder name where Ansible application role files are stored.
	ansibleAppsRolesFolder               = "apps_roles"
	ansibleAppsRolesCurrentVersionFolder = "current"

	// agentCertificatesFolder is the default directory path for storing certificates.
	agentCertificatesFolder = ".certs"
	// agentPrivateKeyFile is the default path for the agent's private key
	agentPrivateKeyFile = "agent.key"
	// agentCSRFile is the default path for the Certificate Signing Request
	agentCSRFile = "agent.csr"
	// agentCertificateFile is the default path for the signed certificate
	agentCertificateFile = "agent.crt"
	// agentCACertificateFile is the default filesystem path for the trusted Certificate Authority (CA) certificate.
	agentCACertificateFile = "ca.crt"

	// gitHubReleasesURL is the default URL for GitHub releases where agent binaries can be downloaded.
	gitHubReleasesURL = "https://github.com/winterflowio/agent/releases/download"
)

// Config holds the application configuration
type Config struct {
	AgentID     string          `json:"agent_id"`
	AgentStatus AgentStatus     `json:"agent_status"`
	Features    map[string]bool `json:"features"`
	// BasePath specifies the root directory used to store application-related files and configurations.
	BasePath string `json:"base_path,omitempty"`
	// LogsPath specifies the directory where log files are stored.
	LogsPath string `json:"logs_path,omitempty"`
	// Orchestrator specifies the orchestration platform or tool used for managing deployments and configurations.
	Orchestrator OrchestratorType `json:"orchestrator,omitempty"`
}

// applyDefaults ensures that all necessary fields have default values if they are empty.
func applyDefaults(cfg *Config) {
	if cfg.AgentStatus == "" {
		cfg.AgentStatus = AgentStatusUnknown
	}
	if cfg.BasePath == "" {
		cfg.BasePath = defaultBasePath
	}
	if cfg.LogsPath == "" {
		cfg.LogsPath = defaultLogsPath
	}
	if cfg.Orchestrator == "" || !isValidOrchestratorType(cfg.Orchestrator) {
		cfg.Orchestrator = defaultOrchestrator
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

func NewConfig() *Config {
	return &Config{
		Features:     make(map[string]bool),
		BasePath:     BasePath,
		LogsPath:     LogsPath,
		Orchestrator: Orchestrator,
	}
}

// LoadConfig loads the configuration from a JSON file
func LoadConfig(configPath string) (*Config, error) {
	config := NewConfig()

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

// WaitUntilReady WaitUntilCompleted waits for the configuration file to exist and have valid content
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

	// Create a copy of the config for saving
	configToSave := *config

	// Filter out features that have default values
	// This ensures we only save features that differ from defaults
	// or features that were explicitly set in the config file
	filteredFeatures := make(map[string]bool)
	for feature, value := range config.Features {
		// Only include the feature if it's not in DefaultFeatureValues
		// or if its value differs from the default
		if defaultValue, exists := DefaultFeatureValues[feature]; !exists || value != defaultValue {
			filteredFeatures[feature] = value
		}
	}
	configToSave.Features = filteredFeatures

	// Marshal config to JSON
	data, err := json.MarshalIndent(configToSave, "", "  ")
	if err != nil {
		return log.Errorf("failed to marshal config: %v", err)
	}

	// Write to file
	if err := os.WriteFile(configPath, data, 0600); err != nil {
		return log.Errorf("failed to write config file: %v", err)
	}

	return nil
}

func (c *Config) GetGRPCServerAddress() string {
	if GRPCServerAddress == "" {
		return defaultGRPCServerAddress
	}
	return GRPCServerAddress
}

func (c *Config) GetAPIBaseURL() string {
	if APIBaseURL == "" {
		return defaultAPIBaseURL
	}
	return APIBaseURL
}

func (c *Config) GetAnsibleFolder() string {
	return ansibleFolder
}

func (c *Config) GetAnsiblePath() string {
	return fmt.Sprintf("%s/%s", c.BasePath, c.GetAnsibleFolder())
}

func (c *Config) GetAnsibleAppsRolesPath() string {
	return fmt.Sprintf("%s/%s", c.GetAnsibleFolder(), ansibleAppsRolesFolder)
}

func (c *Config) GetAnsibleAppRoleCurrentVersionFolder() string {
	return ansibleAppsRolesCurrentVersionFolder
}

func (c *Config) GetCertificateFolder() string {
	return agentCertificatesFolder
}

func (c *Config) GetCertificatePath() string {
	return fmt.Sprintf("%s/%s/%s", c.BasePath, agentCertificatesFolder, agentCertificateFile)
}

func (c *Config) GetPrivateKeyPath() string {
	return fmt.Sprintf("%s/%s/%s", c.BasePath, agentCertificatesFolder, agentPrivateKeyFile)
}

func (c *Config) GetCSRPath() string {
	return fmt.Sprintf("%s/%s/%s", c.BasePath, agentCertificatesFolder, agentCSRFile)
}

func (c *Config) GetCACertificatePath() string {
	return fmt.Sprintf("%s/%s/%s", c.BasePath, agentCertificatesFolder, agentCACertificateFile)
}

func (c *Config) GetCACertificateFile() string {
	return agentCACertificateFile
}

func (c *Config) GetOrchestrator() string {
	return c.Orchestrator.ToString()
}

func isValidOrchestratorType(orchestratorType OrchestratorType) bool {
	return orchestratorType == OrchestratorTypeDockerCompose ||
		orchestratorType == OrchestratorTypeDockerSwarm
}

func (o OrchestratorType) Validate() {
	if !isValidOrchestratorType(o) {
		panic(fmt.Sprintf("invalid orchestrator type: %s, must be one of: %s, %s",
			o, OrchestratorTypeDockerCompose, OrchestratorTypeDockerSwarm))
	}
}

func (o OrchestratorType) ToString() string {
	return string(o)
}

// SetOrchestrator sets the orchestrator type after validating it
func (c *Config) SetOrchestrator(orchestratorType OrchestratorType) error {
	orchestratorType.Validate()
	c.Orchestrator = orchestratorType
	return nil
}

// GetGitHubReleasesURL returns the GitHub releases URL
func (c *Config) GetGitHubReleasesURL() string {
	return gitHubReleasesURL
}
