package get_apps_status

import (
	"context"
	"fmt"
	"winterflow-agent/internal/domain/model"
	"winterflow-agent/internal/domain/repository"
	log "winterflow-agent/pkg/log"
)

// GetAppsStatusQueryHandler handles the GetAppsStatusQuery
type GetAppsStatusQueryHandler struct {
	containerAppRepository repository.ContainerAppRepository
}

// Handle executes the GetAppsStatusQuery and returns the result
func (h *GetAppsStatusQueryHandler) Handle(query GetAppsStatusQuery) (*model.GetAppsStatusResult, error) {
	log.Info("Processing get apps status request")

	// Create a context for the operation
	ctx := context.Background()

	// Get all apps status using the containerAppRepository's GetAppsStatus method
	result, err := h.containerAppRepository.GetAppsStatus(ctx)
	if err != nil {
		log.Error("Error getting apps status", "error", err)
		return nil, fmt.Errorf("failed to get apps status: %w", err)
	}

	log.Info("Retrieved apps status", "apps_count", len(result.Apps))

	return &model.GetAppsStatusResult{
		Apps: result.Apps,
	}, nil
}

// NewGetAppsStatusQueryHandler creates a new GetAppsStatusQueryHandler
func NewGetAppsStatusQueryHandler(orchestrator repository.ContainerAppRepository) *GetAppsStatusQueryHandler {
	return &GetAppsStatusQueryHandler{
		containerAppRepository: orchestrator,
	}
}
