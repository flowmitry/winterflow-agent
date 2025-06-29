package create_network

// CreateNetworkCommand represents a command to create a Docker network.
type CreateNetworkCommand struct {
	NetworkName string // Name of the network to create
}

// Name returns the unique command name for routing on the CQRS bus.
func (c CreateNetworkCommand) Name() string {
	return "CreateNetwork"
}
