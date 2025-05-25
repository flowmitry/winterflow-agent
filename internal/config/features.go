package config

const (
	FeatureUpdateAgent = "update_agent"
)

// DefaultFeatureValues defines the default values for each feature
var DefaultFeatureValues = map[string]bool{
	FeatureUpdateAgent: true,
}

// IsFeatureEnabled returns the GitHub releases URL defined for agent binaries.
func (c *Config) IsFeatureEnabled(feature string) bool {
	for key, value := range c.Features {
		if feature == key {
			return value
		}
	}
	return false
}
