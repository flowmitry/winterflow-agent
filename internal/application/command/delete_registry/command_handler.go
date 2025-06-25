package delete_registry

import (
	"fmt"
	"winterflow-agent/internal/application/config"
	"winterflow-agent/internal/domain/model"
	"winterflow-agent/internal/domain/repository"
	log "winterflow-agent/pkg/log"
)

// DeleteRegistryHandler handles DeleteRegistryCommand operations.
type DeleteRegistryHandler struct {
	repository repository.DockerRegistryRepository
	config     *config.Config
}

// Handle executes the DeleteRegistryCommand.
func (h *DeleteRegistryHandler) Handle(cmd DeleteRegistryCommand) error {
	// Check if registries feature is disabled
	if h.config != nil && h.config.IsFeatureEnabled(config.FeatureRegistriesDisabled) {
		return log.Errorf("registries operations are disabled by configuration")
	}

	log.Info("Processing delete registry command", "address", cmd.Address)

	if cmd.Address == "" {
		return log.Errorf("registry address is required")
	}

	reg := model.Registry{Address: cmd.Address}
	if err := h.repository.DeleteRegistry(reg); err != nil {
		return fmt.Errorf("failed to delete registry: %w", err)
	}

	log.Info("Registry deleted successfully", "address", cmd.Address)
	return nil
}

// NewDeleteRegistryHandler constructs a DeleteRegistryHandler.
func NewDeleteRegistryHandler(repo repository.DockerRegistryRepository, cfg *config.Config) *DeleteRegistryHandler {
	return &DeleteRegistryHandler{repository: repo, config: cfg}
}
