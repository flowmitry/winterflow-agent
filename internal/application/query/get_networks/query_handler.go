package get_networks

import (
	"fmt"
	"winterflow-agent/internal/domain/model"
	"winterflow-agent/internal/domain/repository"
	log "winterflow-agent/pkg/log"
)

// GetNetworksQueryHandler handles the GetNetworksQuery.
type GetNetworksQueryHandler struct {
	repository repository.DockerNetworkRepository
}

// Handle executes the GetNetworksQuery and returns the list of networks.
func (h *GetNetworksQueryHandler) Handle(query GetNetworksQuery) (*model.GetNetworksResult, error) {
	log.Info("Processing get networks query")

	networks, err := h.repository.GetNetworks()
	if err != nil {
		log.Error("Error getting networks", "error", err)
		return nil, fmt.Errorf("failed to get networks: %w", err)
	}

	log.Info("Retrieved networks", "count", len(networks))

	return &model.GetNetworksResult{Networks: networks}, nil
}

// NewGetNetworksQueryHandler creates a new GetNetworksQueryHandler.
func NewGetNetworksQueryHandler(repo repository.DockerNetworkRepository) *GetNetworksQueryHandler {
	return &GetNetworksQueryHandler{repository: repo}
}
