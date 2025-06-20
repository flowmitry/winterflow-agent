package repository

// AppRepository combines deployment and container operations into a single contract.
//
// It is a thin composition of RunnerRepository (deployment/management commands)
// and ContainerAppRepository (runtime status queries).  Any implementation that
// provides both underlying interfaces automatically satisfies AppRepository.
//
// This approach lets us keep multiple infrastructure-level implementations
// (e.g., Ansible + Docker Compose, Ansible + Docker Swarm) while exposing a
// single dependency for the application layer.
//
// Usage example:
//   var repo repository.AppRepository = application.NewAppRepository(runner, container)
//
// Where `runner` implements RunnerRepository and `container` implements
// ContainerAppRepository.
//
// The embedding inside the interface ensures that all methods from both parent
// interfaces are available on AppRepository without writing any additional
// forwarding methods.

type AppRepository interface {
	RunnerRepository
	ContainerAppRepository
}
