package docker_compose

import (
	"sync"

	"winterflow-agent/internal/application/config"
	"winterflow-agent/internal/domain/repository"

	"github.com/docker/docker/client"
)

// composeRepository implements the AppRepository interface for Docker Compose operations.
//
// The implementation is intentionally split across several files in this package:
//  - repository.go       – struct definition, constructor, simple accessors
//  - status.go           – application status related logic
//  - operations.go       – high-level lifecycle operations (deploy, stop, restart, etc.)
//  - compose_cmd.go      – helpers that wrap `docker compose` CLI invocations
//  - template_utils.go   – helper functions for rendering template files
//  - utils.go            – small utility helpers shared by the other files
//
// Splitting the code in this way keeps each file focused on a single responsibility
// which greatly improves readability and maintainability without changing any
// external behaviour.
//
// NOTE: All methods still belong to the *composeRepository receiver – Go allows
// methods to be declared in any file within the same package.

type composeRepository struct {
	client *client.Client
	mu     sync.RWMutex
	config *config.Config
}

// NewComposeRepository creates a new Docker Compose-backed AppRepository implementation.
func NewComposeRepository(cfg *config.Config, dockerClient *client.Client) repository.AppRepository {
	return &composeRepository{
		client: dockerClient,
		config: cfg,
	}
}

// GetClient returns the underlying Docker client instance.
func (r *composeRepository) GetClient() *client.Client {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.client
}
