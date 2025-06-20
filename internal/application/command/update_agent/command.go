package update_agent

// UpdateAgentCommand represents a command to update the agent to a specific version
type UpdateAgentCommand struct {
	Version string
}

// Name returns the name of the command
func (c UpdateAgentCommand) Name() string {
	return "UpdateAgent"
}
