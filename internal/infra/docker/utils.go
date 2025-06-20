package docker

import (
	"strings"
	"winterflow-agent/internal/domain/model"

	"github.com/docker/docker/api/types/container"
)

// MapDockerStateToContainerStatus maps Docker container state to ContainerStatusCode
func MapDockerStateToContainerStatus(state string) model.ContainerStatusCode {
	switch strings.ToLower(state) {
	case "running":
		return model.ContainerStatusActive
	case "exited", "stopped":
		return model.ContainerStatusStopped
	case "restarting":
		return model.ContainerStatusRestarting
	case "paused":
		return model.ContainerStatusIdle
	case "dead", "oomkilled":
		return model.ContainerStatusProblematic
	default:
		return model.ContainerStatusUnknown
	}
}

// MapDockerPortsToContainerPorts converts Docker ports to ContainerPort slice
func MapDockerPortsToContainerPorts(dockerPorts []container.Port) []model.ContainerPort {
	var ports []model.ContainerPort
	for _, dockerPort := range dockerPorts {
		if dockerPort.PublicPort > 0 {
			ports = append(ports, model.ContainerPort{
				Port:     int(dockerPort.PublicPort),
				Protocol: dockerPort.Type,
			})
		}
	}
	return ports
}
