package orchestrator

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"winterflow-agent/internal/config"
	"winterflow-agent/internal/winterflow/models"
	log "winterflow-agent/pkg/log"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
)

// composeRepository implements the Repository interface for Docker Compose
type composeRepository struct {
	client *client.Client
	mu     sync.RWMutex
	config *config.Config
}

// NewComposeRepository creates a new Docker Compose repository
func NewComposeRepository(config *config.Config, dockerClient *client.Client) Repository {
	return &composeRepository{
		client: dockerClient,
		config: config,
	}
}

func (r *composeRepository) GetClient() *client.Client {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.client
}

func (r *composeRepository) GetAppStatus(ctx context.Context, appID string) (GetAppStatusResult, error) {
	log.Debug("Getting Docker Compose app status", "app_id", appID)

	// List containers with the app ID label filter
	filterArgs := filters.NewArgs()
	filterArgs.Add("label", fmt.Sprintf("com.docker.compose.project=%s", appID))

	options := container.ListOptions{
		All:     true,
		Filters: filterArgs,
	}

	dockerContainers, err := r.client.ContainerList(ctx, options)
	if err != nil {
		log.Error("Failed to list containers for app", "app_id", appID, "error", err)
		return GetAppStatusResult{}, fmt.Errorf("failed to list containers: %w", err)
	}

	// Create ContainerApp model
	containerApp := &models.ContainerApp{
		ID:         appID,
		Name:       appID, // For Docker Compose, use project name as app name
		Containers: make([]models.Container, 0, len(dockerContainers)),
	}

	if len(dockerContainers) == 0 {
		log.Debug("No containers found for app", "app_id", appID)
		return GetAppStatusResult{App: containerApp}, nil
	}

	// Convert Docker containers to Container models
	for _, dockerContainer := range dockerContainers {
		container := models.Container{
			ID:         dockerContainer.ID,
			Name:       strings.TrimPrefix(dockerContainer.Names[0], "/"), // Remove leading slash
			StatusCode: mapDockerStateToContainerStatus(dockerContainer.State),
			ExitCode:   0, // Docker API doesn't provide exit code in list response
			Ports:      mapDockerPortsToContainerPorts(dockerContainer.Ports),
		}

		// Add error information for problematic containers
		if container.StatusCode == models.ContainerStatusProblematic {
			container.Error = fmt.Sprintf("Container in problematic state: %s", dockerContainer.Status)
		}

		containerApp.Containers = append(containerApp.Containers, container)
	}

	log.Debug("Docker Compose app status retrieved", "app_id", appID, "containers", len(containerApp.Containers))

	return GetAppStatusResult{App: containerApp}, nil
}
