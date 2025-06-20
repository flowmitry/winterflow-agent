package save_app

import (
	"winterflow-agent/internal/domain/model"
)

// SaveAppCommand represents a command to create a new application
type SaveAppCommand struct {
	App *model.App
}

// Name returns the name of the command
func (c SaveAppCommand) Name() string {
	return "SaveApp"
}
