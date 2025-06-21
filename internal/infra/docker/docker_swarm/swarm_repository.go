package docker_swarm

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"winterflow-agent/internal/application/config"
	"winterflow-agent/internal/domain/model"
	"winterflow-agent/internal/domain/repository"
	"winterflow-agent/internal/infra/docker"
	log "winterflow-agent/pkg/log"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
)

// swarmRepository implements the Repository interface for Docker Swarm
type swarmRepository struct {
	client *client.Client
	mu     sync.RWMutex
	config *config.Config
}

// NewSwarmRepository creates a new Docker Swarm repository
func NewSwarmRepository(config *config.Config, dockerClient *client.Client) repository.AppRepository {
	return &swarmRepository{
		client: dockerClient,
		config: config,
	}
}

func (r *swarmRepository) GetClient() *client.Client {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.client
}

func (r *swarmRepository) GetAppStatus(ctx context.Context, appID string) (model.GetAppStatusResult, error) {
	log.Debug("Getting Docker Swarm app status", "app_id", appID)

	// For Docker Swarm, we look for containers with service labels
	// Services in swarm typically have labels like com.docker.swarm.service.name
	filterArgs := filters.NewArgs()
	filterArgs.Add("label", fmt.Sprintf("com.docker.stack.namespace=%s", appID))
	filterArgs.Add("label", fmt.Sprintf("app=%s", appID)) // Common app label

	options := container.ListOptions{
		All:     true,
		Filters: filterArgs,
	}

	dockerContainers, err := r.client.ContainerList(ctx, options)
	if err != nil {
		log.Error("Failed to list containers for swarm app", "app_id", appID, "error", err)
		return model.GetAppStatusResult{App: nil}, fmt.Errorf("failed to list containers: %w", err)
	}

	// Create ContainerApp model
	containerApp := &model.ContainerApp{
		ID:         appID,
		Name:       appID, // For Docker Swarm, use stack name as app name
		Containers: make([]model.Container, 0, len(dockerContainers)),
	}

	if len(dockerContainers) == 0 {
		log.Debug("No containers found for swarm app", "app_id", appID)
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

		// Add service information to container name for Swarm mode
		if len(dockerContainer.Labels) > 0 {
			if serviceName, exists := dockerContainer.Labels["com.docker.swarm.service.name"]; exists {
				container.Name = fmt.Sprintf("%s (service: %s)", container.Name, serviceName)
			}
		}

		containerApp.Containers = append(containerApp.Containers, container)
	}

	log.Debug("Docker Swarm app status retrieved", "app_id", appID, "containers", len(containerApp.Containers))

	return model.GetAppStatusResult{App: containerApp}, nil
}

func (r *swarmRepository) GetAppsStatus(ctx context.Context) (model.GetAppsStatusResult, error) {
	log.Debug("Getting Docker Swarm apps status for all applications")

	// For Docker Swarm, we look for containers with service labels
	// Services in swarm typically have labels like com.docker.swarm.service.name
	filterArgs := filters.NewArgs()
	filterArgs.Add("label", "com.docker.stack.namespace")

	options := container.ListOptions{
		All:     true,
		Filters: filterArgs,
	}

	dockerContainers, err := r.client.ContainerList(ctx, options)
	if err != nil {
		log.Error("Failed to list containers for all swarm apps", "error", err)
		return model.GetAppsStatusResult{}, fmt.Errorf("failed to list containers: %w", err)
	}

	// Also check for containers with generic app labels
	filterArgsApp := filters.NewArgs()
	filterArgsApp.Add("label", "app")

	optionsApp := container.ListOptions{
		All:     true,
		Filters: filterArgsApp,
	}

	dockerContainersApp, err := r.client.ContainerList(ctx, optionsApp)
	if err != nil {
		log.Error("Failed to list containers with app labels", "error", err)
		return model.GetAppsStatusResult{}, fmt.Errorf("failed to list containers with app labels: %w", err)
	}

	// Combine containers from both searches
	allContainers := append(dockerContainers, dockerContainersApp...)

	// Group containers by app ID (stack namespace or app label)
	appContainers := make(map[string][]types.Container)
	for _, dockerContainer := range allContainers {
		var appID string
		var exists bool

		// Check for stack namespace first
		if appID, exists = dockerContainer.Labels["com.docker.stack.namespace"]; exists {
			appContainers[appID] = append(appContainers[appID], dockerContainer)
		} else if appID, exists = dockerContainer.Labels["app"]; exists {
			// Check for generic app label
			appContainers[appID] = append(appContainers[appID], dockerContainer)
		}
	}

	// Create ContainerApp models for each app
	var apps []*model.ContainerApp
	for appID, containers := range appContainers {
		containerApp := &model.ContainerApp{
			ID:         appID,
			Name:       appID, // For Docker Swarm, use stack name as app name
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

			// Add service information to container name for Swarm mode
			if len(dockerContainer.Labels) > 0 {
				if serviceName, exists := dockerContainer.Labels["com.docker.swarm.service.name"]; exists {
					container.Name = fmt.Sprintf("%s (service: %s)", container.Name, serviceName)
				}
			}

			containerApp.Containers = append(containerApp.Containers, container)
		}

		apps = append(apps, containerApp)
	}

	log.Debug("Docker Swarm apps status retrieved", "apps_count", len(apps))

	return model.GetAppsStatusResult{Apps: apps}, nil
}

func (r *swarmRepository) DeployApp(appID, appVersion string) error {
	return fmt.Errorf("DeployApp is not implemented for docker_swarm orchestrator")
}

func (r *swarmRepository) StopApp(appID string) error {
	return fmt.Errorf("StopApp is not implemented for docker_swarm orchestrator")
}

func (r *swarmRepository) RestartApp(appID, appVersion string) error {
	return fmt.Errorf("RestartApp is not implemented for docker_swarm orchestrator")
}

func (r *swarmRepository) UpdateApp(appID string) error {
	return fmt.Errorf("UpdateApp is not implemented for docker_swarm orchestrator")
}

func (r *swarmRepository) DeleteApp(appID string) error {
	return fmt.Errorf("DeleteApp is not implemented for docker_swarm orchestrator")
}
