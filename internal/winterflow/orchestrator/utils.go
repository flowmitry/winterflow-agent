package orchestrator

import (
	"strings"
	"winterflow-agent/internal/winterflow/models"

	"github.com/docker/docker/api/types/container"
)

// mapDockerStateToContainerStatus maps Docker container state to ContainerStatusCode
func mapDockerStateToContainerStatus(state string) models.ContainerStatusCode {
	switch strings.ToLower(state) {
	case "running":
		return models.ContainerStatusActive
	case "exited", "stopped":
		return models.ContainerStatusStopped
	case "restarting":
		return models.ContainerStatusRestarting
	case "paused":
		return models.ContainerStatusIdle
	case "dead", "oomkilled":
		return models.ContainerStatusProblematic
	default:
		return models.ContainerStatusUnknown
	}
}

// mapDockerPortsToContainerPorts converts Docker ports to ContainerPort slice
func mapDockerPortsToContainerPorts(dockerPorts []container.Port) []models.ContainerPort {
	var ports []models.ContainerPort
	for _, dockerPort := range dockerPorts {
		if dockerPort.PublicPort > 0 {
			ports = append(ports, models.ContainerPort{
				Port:     int(dockerPort.PublicPort),
				Protocol: dockerPort.Type,
			})
		}
	}
	return ports
}
