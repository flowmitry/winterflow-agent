package delete_app

// DeleteAppCommand represents a command to delete an application
type DeleteAppCommand struct {
	AppID string
}

// Name returns the name of the command
func (c DeleteAppCommand) Name() string {
	return "DeleteApp"
}
