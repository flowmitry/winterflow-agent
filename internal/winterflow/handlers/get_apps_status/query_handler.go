package get_apps_status

import (
	"context"
	"fmt"
	"winterflow-agent/internal/winterflow/grpc/pb"
	"winterflow-agent/internal/winterflow/models"
	"winterflow-agent/internal/winterflow/orchestrator"
	log "winterflow-agent/pkg/log"

	"github.com/docker/docker/api/types/container"
)

// GetAppsStatusQueryHandler handles the GetAppsStatusQuery
type GetAppsStatusQueryHandler struct {
	orchestrator orchestrator.Repository
}

// Handle executes the GetAppsStatusQuery and returns the result
func (h *GetAppsStatusQueryHandler) Handle(query GetAppsStatusQuery) ([]*pb.AppStatusV1, error) {
	log.Info("Processing get apps status request")

	// Create a context for the operation
	ctx := context.Background()

	// Create a client to get Docker information
	client := h.orchestrator.GetClient()

	// Get the list of all containers
	options := container.ListOptions{
		All: true,
	}
	containers, err := client.ContainerList(ctx, options)
	if err != nil {
		return nil, fmt.Errorf("failed to list containers: %w", err)
	}

	// Group containers by app ID
	appIDs := make(map[string]bool)
	for _, container := range containers {
		// Extract app ID from container labels
		appID, ok := container.Labels["winterflow.app.id"]
		if !ok {
			// Skip containers that don't have the winterflow.app.id label
			continue
		}

		// Add app ID to the set
		appIDs[appID] = true
	}

	var appStatuses []*pb.AppStatusV1

	// Process each app
	for appID := range appIDs {
		// Get the app status
		result, err := h.orchestrator.GetAppStatus(ctx, appID)
		if err != nil {
			log.Error("Error getting app status", "appID", appID, "error", err)
			continue
		}

		// Convert models.Container to pb.ContainerStatusV1
		containerStatuses := convertContainers(result.App.Containers)

		// Create an AppStatusV1 for this app
		appStatus := &pb.AppStatusV1{
			AppId:      appID,
			Containers: containerStatuses,
		}

		appStatuses = append(appStatuses, appStatus)
	}

	return appStatuses, nil
}

// convertContainers converts models.Container to pb.ContainerStatusV1
func convertContainers(containers []models.Container) []*pb.ContainerStatusV1 {
	var result []*pb.ContainerStatusV1

	for _, container := range containers {
		containerStatus := &pb.ContainerStatusV1{
			ContainerId: container.ID,
			Name:        container.Name,
			StatusCode:  convertStatusCode(container.StatusCode),
			ExitCode:    int32(container.ExitCode),
			Error:       container.Error,
		}
		result = append(result, containerStatus)
	}

	return result
}

// convertStatusCode converts models.ContainerStatusCode to pb.ContainerStatusCode
func convertStatusCode(statusCode models.ContainerStatusCode) pb.ContainerStatusCode {
	switch statusCode {
	case models.ContainerStatusActive:
		return pb.ContainerStatusCode_CONTAINER_STATUS_CODE_ACTIVE
	case models.ContainerStatusIdle:
		return pb.ContainerStatusCode_CONTAINER_STATUS_CODE_IDLE
	case models.ContainerStatusRestarting:
		return pb.ContainerStatusCode_CONTAINER_STATUS_CODE_RESTARTING
	case models.ContainerStatusProblematic:
		return pb.ContainerStatusCode_CONTAINER_STATUS_CODE_PROBLEMATIC
	case models.ContainerStatusStopped:
		return pb.ContainerStatusCode_CONTAINER_STATUS_CODE_STOPPED
	default:
		return pb.ContainerStatusCode_CONTAINER_STATUS_CODE_UNKNOWN
	}
}

// NewGetAppsStatusQueryHandler creates a new GetAppsStatusQueryHandler
func NewGetAppsStatusQueryHandler(orchestrator orchestrator.Repository) *GetAppsStatusQueryHandler {
	return &GetAppsStatusQueryHandler{
		orchestrator: orchestrator,
	}
}
