package registry

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"

	"winterflow-agent/internal/domain/model"
	"winterflow-agent/internal/domain/repository"
	"winterflow-agent/pkg/log"
)

// dockerRegistryRepository provides a Docker-CLI backed implementation of the
// repository.DockerRegistryRepository interface. All operations are executed
// by shelling-out to the `docker` binary that must be available on the host.
//
// The implementation purposefully keeps a very small surface area – only the
// logic that is required by the interface – in order to stay maintainable and
// easy to reason about. Concurrency-safety is achieved with a simple mutex as
// all operations modify the same Docker configuration on disk.
//
// NOTE: We **only** rely on the Go standard library and the custom logging
// package as required by the project rules.
type dockerRegistryRepository struct {
	mu sync.Mutex
}

// Assert that *dockerRegistryRepository implements repository.DockerRegistryRepository.
var _ repository.DockerRegistryRepository = (*dockerRegistryRepository)(nil)

// NewDockerRegistryRepository instantiates a new Docker registry repository.
func NewDockerRegistryRepository() repository.DockerRegistryRepository {
	return &dockerRegistryRepository{}
}

// GetRegistries parses the local Docker configuration (~/.docker/config.json)
// and returns the list of configured registries. If the configuration file can
// not be found an empty slice is returned.
func (r *dockerRegistryRepository) GetRegistries() ([]model.Registry, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	cfgPath, err := getDockerConfigPath()
	if err != nil {
		return nil, err
	}

	// The user may not have logged-in to any registry yet – treat missing file
	// as "no registries configured" instead of an error.
	if _, err := os.Stat(cfgPath); err != nil {
		if os.IsNotExist(err) {
			return []model.Registry{}, nil
		}
		return nil, fmt.Errorf("failed to stat docker config: %w", err)
	}

	raw, err := os.ReadFile(cfgPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read docker config: %w", err)
	}

	// We only care about the auths section which maps registry addresses to
	// arbitrary objects (credentials/helpful fields). A loosely-typed struct is
	// sufficient for our needs.
	var cfg struct {
		Auths map[string]json.RawMessage `json:"auths"`
	}
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse docker config: %w", err)
	}

	registries := make([]model.Registry, 0, len(cfg.Auths))
	for addr := range cfg.Auths {
		registries = append(registries, model.Registry{Address: addr})
	}

	return registries, nil
}

// CreateRegistry logs-in to a Docker registry using the provided credentials.
// Internally this simply shells-out to `docker login`. Password is passed via
// STDIN to avoid exposing it via the process list.
func (r *dockerRegistryRepository) CreateRegistry(registry model.Registry, username, password string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	cmd := exec.Command("docker", "login", registry.Address, "--username", username, "--password-stdin")
	cmd.Stdin = stringReader(password)

	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Error("[Registry] docker login failed", "address", registry.Address, "error", err, "output", string(output))
		return fmt.Errorf("docker login failed: %w", err)
	}

	log.Info("[Registry] login successful", "address", registry.Address)
	return nil
}

// DeleteRegistry logs-out from a Docker registry (`docker logout`).
func (r *dockerRegistryRepository) DeleteRegistry(address string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	cmd := exec.Command("docker", "logout", address)
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Error("[Registry] docker logout failed", "address", address, "error", err, "output", string(output))
		return fmt.Errorf("docker logout failed: %w", err)
	}

	log.Info("[Registry] logout successful", "address", address)
	return nil
}

// getDockerConfigPath resolves the path to the Docker configuration file
// (~/.docker/config.json) while taking the DOCKER_CONFIG environment variable
// into account.
func getDockerConfigPath() (string, error) {
	dockerCfgDir := os.Getenv("DOCKER_CONFIG")
	if dockerCfgDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("unable to determine home directory: %w", err)
		}
		dockerCfgDir = filepath.Join(home, ".docker")
	}
	return filepath.Join(dockerCfgDir, "config.json"), nil
}

// stringReader is a tiny helper that returns an *os.File like reader for the
// provided string making it convenient to pipe data into exec.Command stdin.
func stringReader(s string) *os.File {
	// Using a temporary pipe is slightly more efficient than bytes.NewBuffer.
	r, w, _ := os.Pipe()

	// Create a buffered channel to ensure the goroutine doesn't block
	// if the command exits before reading all data
	done := make(chan struct{}, 1)

	// Create a goroutine that will write the data to the pipe
	// and then close the write end
	go func() {
		defer w.Close()

		// Write the data to the pipe
		_, err := w.Write([]byte(s))
		if err != nil {
			log.Debug("Failed to write to pipe", "error", err)
		}

		// Signal that we're done
		done <- struct{}{}
	}()

	// Return the read end of the pipe
	return r
}
