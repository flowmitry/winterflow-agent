package model

type ContainerAppStatusCode int8

type ContainerStatusCode int8

const (
	ContainerStatusUnknown     ContainerStatusCode = 0
	ContainerStatusActive      ContainerStatusCode = 1
	ContainerStatusIdle        ContainerStatusCode = 2
	ContainerStatusRestarting  ContainerStatusCode = 3
	ContainerStatusProblematic ContainerStatusCode = 4
	ContainerStatusStopped     ContainerStatusCode = 5
)

type ContainerApp struct {
	ID         string      `json:"id"`
	Name       string      `json:"name"`
	Containers []Container `json:"containers"`
}

type Container struct {
	ID         string              `json:"id"`
	Name       string              `json:"name"`
	StatusCode ContainerStatusCode `json:"status_code"`
	ExitCode   int                 `json:"exit_code"`
	Error      string              `json:"error,omitempty"`
	Ports      []ContainerPort     `json:"ports,omitempty"`
}

type ContainerPort struct {
	Port     int    `json:"port"`
	Protocol string `json:"protocol"`
}
