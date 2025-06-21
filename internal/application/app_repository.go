package application

import (
	"github.com/docker/docker/client"
	"winterflow-agent/internal/application/config"
	pkgconfig "winterflow-agent/internal/application/config"
	"winterflow-agent/internal/domain/repository"
	"winterflow-agent/internal/infra/docker/docker_compose"
	"winterflow-agent/internal/infra/docker/docker_swarm"
	"winterflow-agent/pkg/log"
)

func NewAppRepository(config *config.Config) repository.AppRepository {
	dockerClient, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		log.Fatal("Failed to create Docker client", "error", err)
	}

	switch config.GetOrchestrator() {
	case pkgconfig.OrchestratorTypeDockerCompose.ToString():
		return docker_compose.NewComposeRepository(config, dockerClient)
	case pkgconfig.OrchestratorTypeDockerSwarm.ToString():
		return docker_swarm.NewSwarmRepository(config, dockerClient)
	default:
		log.Warn("Unknown orchestrator type, defaulting to Docker Compose", "orchestrator", config.Orchestrator)
		return docker_compose.NewComposeRepository(config, dockerClient)
	}
}
