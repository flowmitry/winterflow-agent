package delete_registry

// DeleteRegistryCommand represents a command to remove/logout from a Docker registry.
type DeleteRegistryCommand struct {
	Address string
}

// Name returns unique command name.
func (c DeleteRegistryCommand) Name() string {
	return "DeleteRegistry"
}
