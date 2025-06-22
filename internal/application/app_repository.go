package application

import (
	"winterflow-agent/internal/application/config"
	pkgconfig "winterflow-agent/internal/application/config"
	"winterflow-agent/internal/domain/repository"
	"winterflow-agent/internal/infra/orchestrator/docker_compose"
	"winterflow-agent/pkg/log"

	"github.com/docker/docker/client"
)

func NewAppRepository(config *config.Config) repository.AppRepository {
	dockerClient, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		log.Fatal("Failed to create Docker client", "error", err)
	}

	switch config.GetOrchestrator() {
	case pkgconfig.OrchestratorTypeDockerCompose.ToString():
		return docker_compose.NewComposeRepository(config, dockerClient)
	default:
		log.Warn("Unknown orchestrator type, defaulting to Docker Compose", "orchestrator", config.Orchestrator)
		return docker_compose.NewComposeRepository(config, dockerClient)
	}
}
