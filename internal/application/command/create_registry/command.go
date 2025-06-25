package create_registry

// CreateRegistryCommand represents a command to add/login to a Docker registry.
type CreateRegistryCommand struct {
	Address  string // Registry hostname/address
	Username string
	Password string
}

// Name returns the unique command name for routing on the CQRS bus.
func (c CreateRegistryCommand) Name() string {
	return "CreateRegistry"
}
