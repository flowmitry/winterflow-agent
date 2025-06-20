package application

import "winterflow-agent/internal/domain/repository"
import "winterflow-agent/internal/infra/ansible"
import pkgconfig "winterflow-agent/internal/config"

func NewRunnerRepository(config *pkgconfig.Config) repository.RunnerRepository {
	return ansible.NewRepository(config)
}
