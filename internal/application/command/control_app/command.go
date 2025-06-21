package control_app

// AppAction represents the action to perform on an application
type AppAction int

const (
	// AppActionStop stops the application
	AppActionStop AppAction = iota
	// AppActionStart starts the application
	AppActionStart
	// AppActionRestart restarts the application
	AppActionRestart
	// AppActionUpdate updates the application
	AppActionUpdate
)

// ControlAppCommand represents a command to control the state of an application
type ControlAppCommand struct {
	AppID      string
	AppVersion uint32
	Action     AppAction
}

// Name returns the name of the command
func (c ControlAppCommand) Name() string {
	return "ControlApp"
}
