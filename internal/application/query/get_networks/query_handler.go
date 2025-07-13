package get_networks

import (
	"fmt"
	"winterflow-agent/internal/application/config"
	"winterflow-agent/internal/domain/dto"
	"winterflow-agent/internal/domain/repository"
	"winterflow-agent/pkg/log"
)

// GetNetworksQueryHandler handles the GetNetworksQuery.
type GetNetworksQueryHandler struct {
	repository repository.DockerNetworkRepository
	config     *config.Config
}

// Handle executes the GetNetworksQuery and returns the list of networks.
func (h *GetNetworksQueryHandler) Handle(query GetNetworksQuery) (*dto.GetNetworksResult, error) {
	// Check if networks feature is disabled
	if h.config != nil && !h.config.IsFeatureEnabled(config.FeatureDockerNetworks) {
		return nil, log.Errorf("networks operations are disabled by configuration")
	}

	log.Info("Processing get networks query")

	networks, err := h.repository.GetNetworks()
	if err != nil {
		log.Error("Error getting networks", "error", err)
		return nil, fmt.Errorf("failed to get networks: %w", err)
	}

	log.Info("Retrieved networks", "count", len(networks))

	return &dto.GetNetworksResult{Networks: networks}, nil
}

// NewGetNetworksQueryHandler creates a new GetNetworksQueryHandler.
func NewGetNetworksQueryHandler(repo repository.DockerNetworkRepository, cfg *config.Config) *GetNetworksQueryHandler {
	return &GetNetworksQueryHandler{repository: repo, config: cfg}
}
