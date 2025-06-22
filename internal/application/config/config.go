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

const (
	// AgentStatusRegistered indicates the agent is registered with the server
	AgentStatusRegistered AgentStatus = "registered"
	// AgentStatusPending indicates the agent registration is pending
	AgentStatusPending AgentStatus = "pending"
	// AgentStatusUnknown indicates the agent status is unknown
	AgentStatusUnknown AgentStatus = "unknown"
)

// OrchestratorType represents the orchestrator
type OrchestratorType string

const (
	OrchestratorTypeDockerCompose OrchestratorType = "docker_compose"
	defaultOrchestrator                            = OrchestratorTypeDockerCompose
)

var (
	grpcServerAddress string
	apiBaseURL        string
	basePath          string
	logsPath          string
	orchestrator      OrchestratorType
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

	// Apps folders
	appsFolder               = "apps"
	appsTemplatesFolder      = "apps_templates"
	appsCurrentVersionFolder = "00001"

	// certificatesFolder is the default directory path for storing certificates.
	certificatesFolder = ".certs"
	// agentPrivateKeyFile is the default path for the agent's private key
	agentPrivateKeyFile = "agent.key"
	// agentCSRFile is the default path for the Certificate Signing Request
	agentCSRFile = "agent.csr"
	// agentCertificateFile is the default path for the signed certificate
	agentCertificateFile = "agent.crt"
	// agentCACertificateFile is the default filesystem path for the trusted Certificate Authority (CA) certificate.
	agentCACertificateFile = "ca.crt"

	// gitHubReleasesURL is the default URL for GitHub releases where agent binaries can be downloaded.
	gitHubReleasesURL = "https://github.com/flowmitry/winterflow-agent/releases/download"
)

// Config holds the application configuration
type Config struct {
	AgentID     string          `json:"agent_id"`
	AgentStatus AgentStatus     `json:"agent_status"`
	Features    map[string]bool `json:"features"`
	// Email specifies the email address associated with the agent
	Email string `json:"email,omitempty"`
	// BasePath specifies the root directory used to store application-related files and configurations.
	BasePath string `json:"base_path,omitempty"`
	// LogsPath specifies the directory where log files are stored.
	LogsPath string `json:"logs_path,omitempty"`
	// LogLevel specifies the minimum log level to output (debug, info, warn, error).
	LogLevel string `json:"log_level,omitempty"`
	// Orchestrator specifies the orchestration platform or tool used for managing deployments and configurations.
	Orchestrator OrchestratorType `json:"orchestrator,omitempty"`
	// CertificatesFolder specifies the directory where certificate files are stored.
	CertificatesFolder string `json:"certificates_folder,omitempty"`
}

// prepareConfig ensures the configuration is valid by applying defaults and validating features
func prepareConfig(cfg *Config) {
	// Apply defaults for required fields
	if cfg.AgentStatus == "" {
		cfg.AgentStatus = AgentStatusUnknown
	}
	if cfg.BasePath == "" {
		cfg.BasePath = defaultBasePath
	}
	if cfg.LogsPath == "" {
		cfg.LogsPath = defaultLogsPath
	}
	if cfg.LogLevel == "" {
		cfg.LogLevel = "info"
	}
	if cfg.Orchestrator == "" || !isValidOrchestratorType(cfg.Orchestrator) {
		cfg.Orchestrator = defaultOrchestrator
	}
	if cfg.CertificatesFolder == "" {
		cfg.CertificatesFolder = certificatesFolder
	}

	// Validate and merge features
	cfg.Features = validateAndMergeFeatures(cfg.Features)
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
	config := &Config{
		Features: make(map[string]bool),
	}

	// Apply build-time overrides or defaults
	if basePath != "" {
		config.BasePath = basePath
	} else {
		config.BasePath = defaultBasePath
	}

	if logsPath != "" {
		config.LogsPath = logsPath
	} else {
		config.LogsPath = defaultLogsPath
	}

	if orchestrator != "" {
		config.Orchestrator = orchestrator
	} else {
		config.Orchestrator = defaultOrchestrator
	}

	return config
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
				// Prepare the config (apply defaults and validate features)
				prepareConfig(config)
				return config, nil
			}
		}
	}

	prepareConfig(config)
	return config, nil
}

// WaitUntilReady WaitUntilCompleted waits for the configuration file to exist and have valid content
func WaitUntilReady(configPath string) (*Config, error) {
	fmt.Printf("\nWaiting for valid configuration file with registered status at %s...", configPath)
	for {
		if _, err := os.Stat(configPath); err == nil {
			// Try to read and validate the config
			data, err := os.ReadFile(configPath)
			if err == nil {
				var config Config // Start with an empty config
				if err := json.Unmarshal(data, &config); err == nil {
					// Check if required fields are filled and agent is registered
					if config.AgentID != "" && config.AgentStatus == AgentStatusRegistered {
						prepareConfig(&config)
						return &config, nil
					}
				}
			}
		}
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

	// Prepare the config (apply defaults and validate features)
	prepareConfig(config)

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
	if grpcServerAddress == "" {
		return defaultGRPCServerAddress
	}
	return grpcServerAddress
}

func (c *Config) GetAPIBaseURL() string {
	if apiBaseURL == "" {
		return defaultAPIBaseURL
	}
	return apiBaseURL
}

func (c *Config) GetLogsPath() string {
	return c.LogsPath
}

func (c *Config) GetAppsPath() string {
	return c.buildPath(appsFolder)
}

func (c *Config) GetAppsTemplatesPath() string {
	return c.buildPath(appsTemplatesFolder)
}

func (c *Config) GetAppsCurrentVersionFolder() string {
	return appsCurrentVersionFolder
}

func (c *Config) GetCertificatesPath() string {
	return c.buildPath(c.GetCertificatesFolder())
}

func (c *Config) GetCertificatesDefaultFolder() string {
	return certificatesFolder
}

func (c *Config) GetCertificatesFolder() string {
	if c.CertificatesFolder == "" {
		return certificatesFolder
	}
	return c.CertificatesFolder
}

// buildPath constructs a file path from base path and components
func (c *Config) buildPath(components ...string) string {
	parts := append([]string{c.BasePath}, components...)
	return filepath.Join(parts...)
}

func (c *Config) GetCertificatePath() string {
	return c.buildPath(c.GetCertificatesFolder(), agentCertificateFile)
}

func (c *Config) GetPrivateKeyPath() string {
	return c.buildPath(c.GetCertificatesFolder(), agentPrivateKeyFile)
}

func (c *Config) GetCSRPath() string {
	return c.buildPath(c.GetCertificatesFolder(), agentCSRFile)
}

func (c *Config) GetCACertificatePath() string {
	return c.buildPath(c.GetCertificatesFolder(), agentCACertificateFile)
}

func (c *Config) GetCACertificateFile() string {
	return agentCACertificateFile
}

func (c *Config) GetOrchestrator() string {
	return c.Orchestrator.ToString()
}

func isValidOrchestratorType(orchestratorType OrchestratorType) bool {
	return orchestratorType == OrchestratorTypeDockerCompose
}

func (o OrchestratorType) Validate() {
	if !isValidOrchestratorType(o) {
		panic(fmt.Sprintf("invalid orchestrator type: %s, must be one of: %s, %s",
			o, OrchestratorTypeDockerCompose))
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

// GetEmail returns the email address associated with the agent
func (c *Config) GetEmail() string {
	return c.Email
}
