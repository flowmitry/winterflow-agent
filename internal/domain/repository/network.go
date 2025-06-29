package repository

import (
	"winterflow-agent/internal/domain/model"
)

type DockerNetworkRepository interface {
	GetNetworks() ([]model.Network, error)

	CreateNetwork(network model.Network) error

	DeleteNetwork(name string) error
}
