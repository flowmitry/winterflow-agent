package save_app

import (
	"winterflow-agent/internal/winterflow/grpc/pb"
)

// SaveAppCommand represents a command to create a new application
type SaveAppCommand struct {
	Request *pb.SaveAppRequestV1
}

// Name returns the name of the command
func (c SaveAppCommand) Name() string {
	return "SaveApp"
}
