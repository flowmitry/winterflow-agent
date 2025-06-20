package application

import (
	"github.com/docker/docker/client"
	"winterflow-agent/internal/application/config"
	pkgconfig "winterflow-agent/internal/application/config"
	"winterflow-agent/internal/domain/repository"
	"winterflow-agent/internal/infra/ansible"
	"winterflow-agent/internal/infra/docker/docker_compose"
	"winterflow-agent/internal/infra/docker/docker_swarm"
	"winterflow-agent/pkg/log"
)

// NewAppRepository returns a repository.AppRepository by composing a
// RunnerRepository and a ContainerAppRepository.  The returned struct embeds
// both repositories, so all their methods are promoted and the composite
// automatically satisfies the AppRepository interface.
func NewAppRepository(config *config.Config) repository.AppRepository {
	return &combinedRepository{
		RunnerRepository:       newRunnerRepository(config),
		ContainerAppRepository: newContainerAppRepository(config),
	}
}

// combinedRepository is an internal adapter that embeds the two concrete
// repositories.  Because embedding promotes method sets, no additional code is
// needed to forward the calls; the compiler does that for us.
type combinedRepository struct {
	repository.RunnerRepository
	repository.ContainerAppRepository
}

func newRunnerRepository(config *pkgconfig.Config) repository.RunnerRepository {
	return ansible.NewRepository(config)
}

func newContainerAppRepository(config *config.Config) repository.ContainerAppRepository {
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
