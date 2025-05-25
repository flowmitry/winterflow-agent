package config

const (
	FeatureUpdateAgent = "update_agent"
)

// DefaultFeatureValues defines the default values for each feature
var DefaultFeatureValues = map[string]bool{
	FeatureUpdateAgent: true,
}

// IsFeatureEnabled checks if a feature is enabled in the configuration.
func (c *Config) IsFeatureEnabled(feature string) bool {
	value, exists := c.Features[feature]
	if !exists {
		// If the feature doesn't exist in the map, check if it has a default value
		defaultValue, hasDefault := DefaultFeatureValues[feature]
		return hasDefault && defaultValue
	}
	return value
}
