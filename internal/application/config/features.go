package config

const (
	FeatureAgentUpdate      = "agent_update"
	FeatureEarlyAccess      = "early_access"
	FeatureDockerRegistries = "docker_registries"
)

// DefaultFeatureValues defines the default values for each feature
var DefaultFeatureValues = map[string]bool{
	FeatureAgentUpdate:      true,
	FeatureEarlyAccess:      false,
	FeatureDockerRegistries: true,
}

// IsFeatureEnabled checks if a feature is enabled in the configuration.
func (c *Config) IsFeatureEnabled(feature string) bool {
	value, exists := c.Features[feature]
	if !exists {
		return DefaultFeatureValues[feature]
	}
	return value
}
