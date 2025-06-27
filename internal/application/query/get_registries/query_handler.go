package get_registries

import (
	"fmt"
	"winterflow-agent/internal/application/config"
	"winterflow-agent/internal/domain/model"
	"winterflow-agent/internal/domain/repository"
	log "winterflow-agent/pkg/log"
)

// GetRegistriesQueryHandler handles the GetRegistriesQuery.
type GetRegistriesQueryHandler struct {
	repository repository.DockerRegistryRepository
	config     *config.Config
}

// Handle executes the GetRegistriesQuery and returns the list of registries.
func (h *GetRegistriesQueryHandler) Handle(query GetRegistriesQuery) (*model.GetRegistriesResult, error) {
	// Check if registries feature is disabled
	if h.config != nil && h.config.IsFeatureEnabled(config.FeatureDockerRegistries) {
		return nil, log.Errorf("registries operations are disabled by configuration")
	}

	log.Info("Processing get registries query")

	registries, err := h.repository.GetRegistries()
	if err != nil {
		log.Error("Error getting registries", "error", err)
		return nil, fmt.Errorf("failed to get registries: %w", err)
	}

	log.Info("Retrieved registries", "count", len(registries))

	return &model.GetRegistriesResult{Registries: registries}, nil
}

// NewGetRegistriesQueryHandler creates a new GetRegistriesQueryHandler.
func NewGetRegistriesQueryHandler(repo repository.DockerRegistryRepository, cfg *config.Config) *GetRegistriesQueryHandler {
	return &GetRegistriesQueryHandler{repository: repo, config: cfg}
}
