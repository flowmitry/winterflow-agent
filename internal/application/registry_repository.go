package application

import (
	"winterflow-agent/internal/domain/repository"
	"winterflow-agent/internal/infra/docker/registry"
)

func NewRegistryRepository() repository.DockerRegistryRepository {
	return registry.NewDockerRegistryRepository()
}
