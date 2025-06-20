package capabilities

import (
	"winterflow-agent/internal/config"
	"winterflow-agent/internal/infra/winterflow/api"
	log "winterflow-agent/pkg/log"
)

// ServerIPCapability represents the IP Address capability
type ServerIPCapability struct {
	ipAddress string
}

// NewServerIPCapability creates a new IP Address capability
func NewServerIPCapability() *ServerIPCapability {
	return &ServerIPCapability{
		ipAddress: "unknown", // Default value
	}
}

// Name returns the name of the capability
func (c *ServerIPCapability) Name() string {
	return CapabilityServerIP
}

// Value returns the value of the capability
func (c *ServerIPCapability) Value() string {
	// If we already have the IP address, return it
	if c.ipAddress != "unknown" {
		return c.ipAddress
	}

	// Otherwise, fetch it from the API
	c.fetchIPAddress()
	return c.ipAddress
}

// fetchIPAddress fetches the IP address from the API
func (c *ServerIPCapability) fetchIPAddress() {
	// Create a new config to get the API base URL
	cfg := config.NewConfig()

	// Create a new API client
	client := api.NewClient(cfg.GetAPIBaseURL())

	// Get the IP address
	ipAddress, err := client.GetIPAddress()
	if err != nil {
		log.Error("Failed to fetch IP address: %v", err)
		return
	}

	// Update the IP address
	c.ipAddress = ipAddress
}
