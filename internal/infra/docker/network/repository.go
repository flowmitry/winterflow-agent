package network

import (
	"context"
	"fmt"
	"sync"

	networktypes "github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"

	"winterflow-agent/internal/domain/model"
	"winterflow-agent/internal/domain/repository"
	"winterflow-agent/pkg/log"
)

// dockerNetworkRepository provides thread-safe methods for managing Docker networks using a Docker client.
type dockerNetworkRepository struct {
	client *client.Client
	mu     sync.RWMutex
}

// Compile-time assertion that *dockerNetworkRepository implements the interface.
var _ repository.DockerNetworkRepository = (*dockerNetworkRepository)(nil)

// NewDockerNetworkRepository creates a new DockerNetworkRepository using the provided Docker client.
// Logs a fatal error and exits the program if the Docker client is nil.
func NewDockerNetworkRepository(dockerClient *client.Client) repository.DockerNetworkRepository {
	if dockerClient == nil {
		log.Fatal("[Network] docker client is nil – repository cannot be created")
	}
	return &dockerNetworkRepository{client: dockerClient}
}

func (r *dockerNetworkRepository) GetNetworks() ([]model.Network, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	ctx := context.Background()
	dockerNetworks, err := r.client.NetworkList(ctx, networktypes.ListOptions{})
	if err != nil {
		log.Error("[Network] failed to list networks", "error", err)
		return nil, fmt.Errorf("list networks: %w", err)
	}

	networks := make([]model.Network, 0, len(dockerNetworks))
	for _, dn := range dockerNetworks {
		networks = append(networks, model.Network{Name: dn.Name})
	}

	return networks, nil
}

func (r *dockerNetworkRepository) CreateNetwork(network model.Network) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	ctx := context.Background()
	_, err := r.client.NetworkCreate(ctx, network.Name, networktypes.CreateOptions{})
	if err != nil {
		log.Error("[Network] failed to create network", "network_name", network.Name, "error", err)
		return fmt.Errorf("create network: %w", err)
	}

	log.Info("[Network] network created", "network_name", network.Name)
	return nil
}

func (r *dockerNetworkRepository) DeleteNetwork(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	ctx := context.Background()
	if err := r.client.NetworkRemove(ctx, name); err != nil {
		log.Error("[Network] failed to remove network", "network_name", name, "error", err)
		return fmt.Errorf("remove network: %w", err)
	}

	log.Info("[Network] network removed", "network_name", name)
	return nil
}
