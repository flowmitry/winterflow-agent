package agent

import (
	"winterflow-agent/pkg/capabilities"
)

func GetCapabilities() *capabilities.CapabilityFactory {
	return capabilities.NewCapabilityFactory()
}
