package config

const (
    FeatureAgentUpdateDisabled = "agent_update_disabled"
    FeatureSendMetricsDisabled = "send_metrics_disabled"
    FeatureEarlyAccessEnabled  = "early_access_enabled"
    FeatureRegistriesDisabled  = "registries_disabled"
)

const (
    FeatureDefaultValue = false
)

// DefaultFeatureValues defines the default values for each feature
var DefaultFeatureValues = map[string]bool{
    FeatureAgentUpdateDisabled: FeatureDefaultValue,
    FeatureEarlyAccessEnabled:  FeatureDefaultValue,
    FeatureSendMetricsDisabled: FeatureDefaultValue,
    FeatureRegistriesDisabled:  FeatureDefaultValue,
}

// IsFeatureEnabled checks if a feature is enabled in the configuration.
func (c *Config) IsFeatureEnabled(feature string) bool {
    value, exists := c.Features[feature]
    if !exists {
        return FeatureDefaultValue
    }
    return value
}
