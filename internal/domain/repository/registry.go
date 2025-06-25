package repository

import (
	"winterflow-agent/internal/domain/model"
)

// DockerRegistryRepository is an interface for managing Docker Registry operations
type DockerRegistryRepository interface {
	GetRegistries() ([]model.Registry, error)

	CreateRegistry(registry model.Registry, username string, password string) error

	DeleteRegistry(registry model.Registry) error
}
