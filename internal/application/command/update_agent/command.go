package update_agent

import (
	"winterflow-agent/internal/infra/winterflow/grpc/pb"
)

// UpdateAgentCommand represents a command to update the agent to a specific version
type UpdateAgentCommand struct {
	Request *pb.UpdateAgentRequestV1
}

// Name returns the name of the command
func (c UpdateAgentCommand) Name() string {
	return "UpdateAgent"
}
