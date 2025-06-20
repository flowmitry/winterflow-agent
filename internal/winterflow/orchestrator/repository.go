package orchestrator

import (
	"context"
	"winterflow-agent/internal/config"
	pkgconfig "winterflow-agent/internal/config"
	"winterflow-agent/internal/winterflow/models"
	log "winterflow-agent/pkg/log"

	"github.com/docker/docker/client"
)

// GetAppStatusResult represents the result of a Docker operation
type GetAppStatusResult struct {
	App *models.ContainerApp
}

// GetAppsStatusResult represents the result of getting all apps status
type GetAppsStatusResult struct {
	Apps []*models.ContainerApp
}

// Repository is an interface for managing Docker operations
type Repository interface {
	// GetClient returns a Docker client for low-level operations
	GetClient() *client.Client

	// GetAppStatus returns the status of a specific application
	GetAppStatus(ctx context.Context, appID string) (GetAppStatusResult, error)

	// GetAppsStatus returns the status of all available applications
	GetAppsStatus(ctx context.Context) (GetAppsStatusResult, error)
}

// NewRepository creates a new Docker repository based on orchestrator configuration
func NewRepository(config *config.Config) Repository {
	dockerClient, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		log.Fatal("Failed to create Docker client", "error", err)
	}

	switch config.GetOrchestrator() {
	case pkgconfig.OrchestratorTypeDockerCompose.ToString():
		return NewComposeRepository(config, dockerClient)
	case pkgconfig.OrchestratorTypeDockerSwarm.ToString():
		return NewSwarmRepository(config, dockerClient)
	default:
		log.Warn("Unknown orchestrator type, defaulting to Docker Compose", "orchestrator", config.Orchestrator)
		return NewComposeRepository(config, dockerClient)
	}
}
