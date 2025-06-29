package application

import (
	"github.com/docker/docker/client"
	"winterflow-agent/internal/domain/repository"
	"winterflow-agent/internal/infra/docker/network"
	"winterflow-agent/pkg/log"
)

func NewNetworkRepository() repository.DockerNetworkRepository {
	dockerClient, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		log.Fatal("Failed to create Docker client", "error", err)
	}
	return network.NewDockerNetworkRepository(dockerClient)
}
