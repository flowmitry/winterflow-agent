package docker_compose

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"winterflow-agent/internal/config"
	"winterflow-agent/internal/domain/model"
	"winterflow-agent/internal/domain/repository"
	"winterflow-agent/internal/infra/docker"
	log "winterflow-agent/pkg/log"

	"github.com/docker/docker/api/types"
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
func NewComposeRepository(config *config.Config, dockerClient *client.Client) repository.ContainerAppRepository {
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

func (r *composeRepository) GetAppStatus(ctx context.Context, appID string) (model.GetAppStatusResult, error) {
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
		return model.GetAppStatusResult{}, fmt.Errorf("failed to list containers: %w", err)
	}

	// Create ContainerApp model
	containerApp := &model.ContainerApp{
		ID:         appID,
		Name:       appID, // For Docker Compose, use project name as app name
		Containers: make([]model.Container, 0, len(dockerContainers)),
	}

	if len(dockerContainers) == 0 {
		log.Debug("No containers found for app", "app_id", appID)
		return model.GetAppStatusResult{App: containerApp}, nil
	}

	// Convert Docker containers to Container models
	for _, dockerContainer := range dockerContainers {
		container := model.Container{
			ID:         dockerContainer.ID,
			Name:       strings.TrimPrefix(dockerContainer.Names[0], "/"), // Remove leading slash
			StatusCode: docker.MapDockerStateToContainerStatus(dockerContainer.State),
			ExitCode:   0, // Docker API doesn't provide exit code in list response
			Ports:      docker.MapDockerPortsToContainerPorts(dockerContainer.Ports),
		}

		// Add error information for problematic containers
		if container.StatusCode == model.ContainerStatusProblematic {
			container.Error = fmt.Sprintf("Container in problematic state: %s", dockerContainer.Status)
		}

		containerApp.Containers = append(containerApp.Containers, container)
	}

	log.Debug("Docker Compose app status retrieved", "app_id", appID, "containers", len(containerApp.Containers))

	return model.GetAppStatusResult{App: containerApp}, nil
}

func (r *composeRepository) GetAppsStatus(ctx context.Context) (model.GetAppsStatusResult, error) {
	log.Debug("Getting Docker Compose apps status for all applications")

	// List all containers with Docker Compose project labels
	filterArgs := filters.NewArgs()
	filterArgs.Add("label", "com.docker.compose.project")

	options := container.ListOptions{
		All:     true,
		Filters: filterArgs,
	}

	dockerContainers, err := r.client.ContainerList(ctx, options)
	if err != nil {
		log.Error("Failed to list containers for all apps", "error", err)
		return model.GetAppsStatusResult{}, fmt.Errorf("failed to list containers: %w", err)
	}

	// Group containers by app ID (compose project)
	appContainers := make(map[string][]types.Container)
	for _, dockerContainer := range dockerContainers {
		if appID, exists := dockerContainer.Labels["com.docker.compose.project"]; exists {
			appContainers[appID] = append(appContainers[appID], dockerContainer)
		}
	}

	// Create ContainerApp models for each app
	var apps []*model.ContainerApp
	for appID, containers := range appContainers {
		containerApp := &model.ContainerApp{
			ID:         appID,
			Name:       appID, // For Docker Compose, use project name as app name
			Containers: make([]model.Container, 0, len(containers)),
		}

		// Convert Docker containers to Container models
		for _, dockerContainer := range containers {
			container := model.Container{
				ID:         dockerContainer.ID,
				Name:       strings.TrimPrefix(dockerContainer.Names[0], "/"), // Remove leading slash
				StatusCode: docker.MapDockerStateToContainerStatus(dockerContainer.State),
				ExitCode:   0, // Docker API doesn't provide exit code in list response
				Ports:      docker.MapDockerPortsToContainerPorts(dockerContainer.Ports),
			}

			// Add error information for problematic containers
			if container.StatusCode == model.ContainerStatusProblematic {
				container.Error = fmt.Sprintf("Container in problematic state: %s", dockerContainer.Status)
			}

			containerApp.Containers = append(containerApp.Containers, container)
		}

		apps = append(apps, containerApp)
	}

	log.Debug("Docker Compose apps status retrieved", "apps_count", len(apps))

	return model.GetAppsStatusResult{Apps: apps}, nil
}
