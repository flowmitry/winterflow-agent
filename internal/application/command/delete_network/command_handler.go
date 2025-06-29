package delete_network

import (
	"fmt"
	"winterflow-agent/internal/application/config"
	"winterflow-agent/internal/domain/repository"
	log "winterflow-agent/pkg/log"
)

// DeleteNetworkHandler integrates with the DockerNetworkRepository to execute DeleteNetworkCommand.
type DeleteNetworkHandler struct {
	repository repository.DockerNetworkRepository
	config     *config.Config
}

// Handle executes the DeleteNetworkCommand.
func (h *DeleteNetworkHandler) Handle(cmd DeleteNetworkCommand) error {
	// Check if networks feature is disabled
	if h.config != nil && !h.config.IsFeatureEnabled(config.FeatureDockerNetworks) {
		return log.Errorf("networks operations are disabled by configuration")
	}

	log.Info("Processing delete network command", "network_name", cmd.NetworkName)

	if cmd.NetworkName == "" {
		return log.Errorf("network name is required")
	}

	if err := h.repository.DeleteNetwork(cmd.NetworkName); err != nil {
		log.Error("Failed to delete network", "network_name", cmd.NetworkName, "error", err)
		return fmt.Errorf("failed to delete network: %w", err)
	}

	log.Info("Network deleted successfully", "network_name", cmd.NetworkName)
	return nil
}

// NewDeleteNetworkHandler returns a configured DeleteNetworkHandler.
func NewDeleteNetworkHandler(repo repository.DockerNetworkRepository, cfg *config.Config) *DeleteNetworkHandler {
	return &DeleteNetworkHandler{repository: repo, config: cfg}
}
