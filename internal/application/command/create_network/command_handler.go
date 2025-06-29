package create_network

import (
	"fmt"
	"winterflow-agent/internal/domain/model"
	"winterflow-agent/internal/domain/repository"
	log "winterflow-agent/pkg/log"
)

// CreateNetworkHandler integrates with the DockerNetworkRepository to execute CreateNetworkCommand.
type CreateNetworkHandler struct {
	repository repository.DockerNetworkRepository
}

// Handle executes the CreateNetworkCommand.
func (h *CreateNetworkHandler) Handle(cmd CreateNetworkCommand) error {
	log.Info("Processing create network command", "network_name", cmd.NetworkName)

	if cmd.NetworkName == "" {
		return log.Errorf("network name is required")
	}

	if err := h.repository.CreateNetwork(model.Network{Name: cmd.NetworkName}); err != nil {
		log.Error("Failed to create network", "network_name", cmd.NetworkName, "error", err)
		return fmt.Errorf("failed to create network: %w", err)
	}

	log.Info("Network created successfully", "network_name", cmd.NetworkName)
	return nil
}

// NewCreateNetworkHandler returns a configured CreateNetworkHandler.
func NewCreateNetworkHandler(repo repository.DockerNetworkRepository) *CreateNetworkHandler {
	return &CreateNetworkHandler{repository: repo}
}
