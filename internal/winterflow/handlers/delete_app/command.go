package delete_app

import (
	"winterflow-agent/internal/winterflow/grpc/pb"
)

// DeleteAppCommand represents a command to delete an application
type DeleteAppCommand struct {
	Request *pb.DeleteAppRequestV1
}

// Name returns the name of the command
func (c DeleteAppCommand) Name() string {
	return "DeleteApp"
}
