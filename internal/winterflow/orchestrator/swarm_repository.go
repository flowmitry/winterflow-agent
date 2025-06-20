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

// swarmRepository implements the Repository interface for Docker Swarm
type swarmRepository struct {
	client *client.Client
	mu     sync.RWMutex
	config *config.Config
}

// NewSwarmRepository creates a new Docker Swarm repository
func NewSwarmRepository(config *config.Config, dockerClient *client.Client) Repository {
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

func (r *swarmRepository) GetAppStatus(ctx context.Context, appID string) (GetAppStatusResult, error) {
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
		return GetAppStatusResult{App: nil}, fmt.Errorf("failed to list containers: %w", err)
	}

	// Create ContainerApp model
	containerApp := &models.ContainerApp{
		ID:         appID,
		Name:       appID, // For Docker Swarm, use stack name as app name
		Containers: make([]models.Container, 0, len(dockerContainers)),
	}

	if len(dockerContainers) == 0 {
		log.Debug("No containers found for swarm app", "app_id", appID)
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

		// Add service information to container name for Swarm mode
		if len(dockerContainer.Labels) > 0 {
			if serviceName, exists := dockerContainer.Labels["com.docker.swarm.service.name"]; exists {
				container.Name = fmt.Sprintf("%s (service: %s)", container.Name, serviceName)
			}
		}

		containerApp.Containers = append(containerApp.Containers, container)
	}

	log.Debug("Docker Swarm app status retrieved", "app_id", appID, "containers", len(containerApp.Containers))

	return GetAppStatusResult{App: containerApp}, nil
}
