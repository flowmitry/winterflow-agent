package delete_network

// DeleteNetworkCommand represents a command to delete a Docker network.
type DeleteNetworkCommand struct {
	NetworkName string // Name of the network to delete
}

// Name returns the unique command name for routing on the CQRS bus.
func (c DeleteNetworkCommand) Name() string {
	return "DeleteNetwork"
}
