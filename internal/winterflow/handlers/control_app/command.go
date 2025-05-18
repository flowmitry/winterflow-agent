package control_app

import (
	"winterflow-agent/internal/winterflow/grpc/pb"
)

// ControlAppCommand represents a command to control the state of an application
type ControlAppCommand struct {
	Request *pb.ControlAppRequestV1
}

// Name returns the name of the command
func (c ControlAppCommand) Name() string {
	return "ControlApp"
}
