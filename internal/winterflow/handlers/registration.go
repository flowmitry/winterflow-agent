package handlers

import (
	"winterflow-agent/internal/winterflow/handlers/get_app"
	"winterflow-agent/internal/winterflow/handlers/save_app"
	"winterflow-agent/pkg/cqrs"
	"winterflow-agent/pkg/log"
)

// RegisterCommandHandlers registers all command handlers with the command bus
func RegisterCommandHandlers(b cqrs.CommandBus) error {
	if err := b.Register(save_app.NewSaveAppHandler()); err != nil {
		return log.Errorf("failed to register save app handler: %v", err)
	}
	return nil
}

// RegisterQueryHandlers registers all query handlers with the query bus
func RegisterQueryHandlers(b cqrs.QueryBus) error {
	if err := b.Register(get_app.NewGetAppQueryHandler()); err != nil {
		return log.Errorf("failed to register get app query handler: %v", err)
	}
	return nil
}
