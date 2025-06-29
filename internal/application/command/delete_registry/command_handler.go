package delete_registry

import (
	"fmt"
	"winterflow-agent/internal/application/config"
	"winterflow-agent/internal/domain/repository"
	"winterflow-agent/pkg/log"
)

// DeleteRegistryHandler handles DeleteRegistryCommand operations.
type DeleteRegistryHandler struct {
	repository repository.DockerRegistryRepository
	config     *config.Config
}

// Handle executes the DeleteRegistryCommand.
func (h *DeleteRegistryHandler) Handle(cmd DeleteRegistryCommand) error {
	// Check if registries feature is disabled
	if h.config != nil && !h.config.IsFeatureEnabled(config.FeatureDockerRegistries) {
		return log.Errorf("registries operations are disabled by configuration")
	}

	log.Info("Processing delete registry command", "address", cmd.Address)

	if cmd.Address == "" {
		return log.Errorf("registry address is required")
	}

	if err := h.repository.DeleteRegistry(cmd.Address); err != nil {
		return fmt.Errorf("failed to delete registry: %w", err)
	}

	log.Info("Registry deleted successfully", "address", cmd.Address)
	return nil
}

// NewDeleteRegistryHandler constructs a DeleteRegistryHandler.
func NewDeleteRegistryHandler(repo repository.DockerRegistryRepository, cfg *config.Config) *DeleteRegistryHandler {
	return &DeleteRegistryHandler{repository: repo, config: cfg}
}
