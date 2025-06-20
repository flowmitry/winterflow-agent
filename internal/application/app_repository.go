package application

import "winterflow-agent/internal/domain/repository"

// NewAppRepository returns a repository.AppRepository by composing a
// RunnerRepository and a ContainerAppRepository.  The returned struct embeds
// both repositories, so all their methods are promoted and the composite
// automatically satisfies the AppRepository interface.
func NewAppRepository(runner repository.RunnerRepository, container repository.ContainerAppRepository) repository.AppRepository {
	return &combinedRepository{
		RunnerRepository:       runner,
		ContainerAppRepository: container,
	}
}

// combinedRepository is an internal adapter that embeds the two concrete
// repositories.  Because embedding promotes method sets, no additional code is
// needed to forward the calls; the compiler does that for us.
type combinedRepository struct {
	repository.RunnerRepository
	repository.ContainerAppRepository
}
