package handlers

import (
	"winterflow-agent/internal/winterflow/handlers/create_app"
	"winterflow-agent/pkg/cqrs"
	"winterflow-agent/pkg/log"
)

// RegisterHandlers registers all command handlers with the command bus
func RegisterHandlers(b cqrs.CommandBus) error {
	if err := b.Register(create_app.NewCreateAppHandler()); err != nil {
		return log.Errorf("failed to register create app handler: %v", err)
	}
	return nil
}
