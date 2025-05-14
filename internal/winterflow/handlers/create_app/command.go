package create_app

import (
	"winterflow-agent/internal/winterflow/grpc/pb"
)

// CreateAppCommand represents a command to create a new application
type CreateAppCommand struct {
	Request *pb.CreateAppRequestV1
}

// Name returns the name of the command
func (c CreateAppCommand) Name() string {
	return "CreateApp"
}
