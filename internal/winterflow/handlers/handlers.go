package handlers

import (
	"winterflow-agent/internal/config"
	configconst "winterflow-agent/internal/config"
	"winterflow-agent/internal/winterflow/ansible"
	"winterflow-agent/internal/winterflow/handlers/control_app"
	"winterflow-agent/internal/winterflow/handlers/delete_app"
	"winterflow-agent/internal/winterflow/handlers/get_app"
	"winterflow-agent/internal/winterflow/handlers/save_app"
	"winterflow-agent/pkg/cqrs"
	"winterflow-agent/pkg/log"
)

// RegisterCommandHandlers registers all command handlers with the command bus
func RegisterCommandHandlers(b cqrs.CommandBus, config *config.Config) error {
	ansibleClient := ansible.NewAnsibleClient(config)
	if err := b.Register(save_app.NewSaveAppHandler(config.GetAnsibleAppsRolesPath(), configconst.AnsibleAppsRolesCurrentVersionFolder, config.PrivateKeyPath)); err != nil {
		return log.Errorf("failed to register save app handler: %v", err)
	}

	if err := b.Register(delete_app.NewDeleteAppHandler(config.GetAnsibleAppsRolesPath())); err != nil {
		return log.Errorf("failed to register delete app handler: %v", err)
	}

	if err := b.Register(control_app.NewControlAppHandler(&ansibleClient, config.GetAnsibleAppsRolesPath())); err != nil {
		return log.Errorf("failed to register control app handler: %v", err)
	}

	return nil
}

// RegisterQueryHandlers registers all query handlers with the query bus
func RegisterQueryHandlers(b cqrs.QueryBus, config *config.Config) error {
	if err := b.Register(get_app.NewGetAppQueryHandler(config.GetAnsibleAppsRolesPath(), configconst.AnsibleAppsRolesCurrentVersionFolder)); err != nil {
		return log.Errorf("failed to register get app query handler: %v", err)
	}
	return nil
}
