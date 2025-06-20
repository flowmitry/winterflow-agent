package get_apps_status

import (
	"context"
	"fmt"
	"winterflow-agent/internal/domain/model"
	"winterflow-agent/internal/domain/repository"
	"winterflow-agent/internal/infra/winterflow/grpc/pb"
	log "winterflow-agent/pkg/log"
)

// GetAppsStatusQueryHandler handles the GetAppsStatusQuery
type GetAppsStatusQueryHandler struct {
	containerAppRepository repository.ContainerAppRepository
}

// Handle executes the GetAppsStatusQuery and returns the result
func (h *GetAppsStatusQueryHandler) Handle(query GetAppsStatusQuery) ([]*pb.AppStatusV1, error) {
	log.Info("Processing get apps status request")

	// Create a context for the operation
	ctx := context.Background()

	// Get all apps status using the containerAppRepository's GetAppsStatus method
	result, err := h.containerAppRepository.GetAppsStatus(ctx)
	if err != nil {
		log.Error("Error getting apps status", "error", err)
		return nil, fmt.Errorf("failed to get apps status: %w", err)
	}

	var appStatuses []*pb.AppStatusV1

	// Process each app from the result
	for _, app := range result.Apps {
		// Convert model.Container to pb.ContainerStatusV1
		containerStatuses := convertContainers(app.Containers)

		// Create an AppStatusV1 for this app
		appStatus := &pb.AppStatusV1{
			AppId:      app.ID,
			Containers: containerStatuses,
		}

		appStatuses = append(appStatuses, appStatus)
	}

	log.Info("Retrieved apps status", "apps_count", len(appStatuses))

	return appStatuses, nil
}

// convertContainers converts model.Container to pb.ContainerStatusV1
func convertContainers(containers []model.Container) []*pb.ContainerStatusV1 {
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

// convertStatusCode converts model.ContainerStatusCode to pb.ContainerStatusCode
func convertStatusCode(statusCode model.ContainerStatusCode) pb.ContainerStatusCode {
	switch statusCode {
	case model.ContainerStatusActive:
		return pb.ContainerStatusCode_CONTAINER_STATUS_CODE_ACTIVE
	case model.ContainerStatusIdle:
		return pb.ContainerStatusCode_CONTAINER_STATUS_CODE_IDLE
	case model.ContainerStatusRestarting:
		return pb.ContainerStatusCode_CONTAINER_STATUS_CODE_RESTARTING
	case model.ContainerStatusProblematic:
		return pb.ContainerStatusCode_CONTAINER_STATUS_CODE_PROBLEMATIC
	case model.ContainerStatusStopped:
		return pb.ContainerStatusCode_CONTAINER_STATUS_CODE_STOPPED
	default:
		return pb.ContainerStatusCode_CONTAINER_STATUS_CODE_UNKNOWN
	}
}

// NewGetAppsStatusQueryHandler creates a new GetAppsStatusQueryHandler
func NewGetAppsStatusQueryHandler(orchestrator repository.ContainerAppRepository) *GetAppsStatusQueryHandler {
	return &GetAppsStatusQueryHandler{
		containerAppRepository: orchestrator,
	}
}
