package rename_app

// RenameAppCommand represents a command to rename an existing application.
// It contains the target application ID and the new desired name.
// The new name must be unique across all applications.
type RenameAppCommand struct {
	AppID   string
	AppName string
}

// Name returns a unique identifier of the command used by the CQRS bus.
func (c RenameAppCommand) Name() string {
	return "RenameApp"
}
