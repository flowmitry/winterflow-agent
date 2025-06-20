package handlers

import (
	"winterflow-agent/internal/config"
	"winterflow-agent/internal/winterflow/ansible"
	"winterflow-agent/internal/winterflow/handlers/control_app"
	"winterflow-agent/internal/winterflow/handlers/delete_app"
	"winterflow-agent/internal/winterflow/handlers/get_app"
	"winterflow-agent/internal/winterflow/handlers/get_apps_status"
	"winterflow-agent/internal/winterflow/handlers/save_app"
	"winterflow-agent/internal/winterflow/handlers/update_agent"
	"winterflow-agent/internal/winterflow/orchestrator"
	"winterflow-agent/pkg/cqrs"
	"winterflow-agent/pkg/log"
)

// RegisterCommandHandlers registers all command handlers with the command bus
func RegisterCommandHandlers(b cqrs.CommandBus, config *config.Config, ansible ansible.Repository) error {
	if err := b.Register(save_app.NewSaveAppHandler(config.GetAnsibleAppsRolesPath(), config.GetAnsibleAppRoleCurrentVersionFolder(), config.GetPrivateKeyPath())); err != nil {
		return log.Errorf("failed to register save app handler: %v", err)
	}

	if err := b.Register(delete_app.NewDeleteAppHandler(ansible, config.GetAnsibleAppsRolesPath(), config.GetAnsibleAppRoleCurrentVersionFolder())); err != nil {
		return log.Errorf("failed to register delete app handler: %v", err)
	}

	if err := b.Register(control_app.NewControlAppHandler(ansible, config.GetAnsibleAppsRolesPath(), config.GetAnsibleAppRoleCurrentVersionFolder())); err != nil {
		return log.Errorf("failed to register control app handler: %v", err)
	}

	if err := b.Register(update_agent.NewUpdateAgentHandler(config)); err != nil {
		return log.Errorf("failed to register update agent handler: %v", err)
	}

	return nil
}

// RegisterQueryHandlers registers all query handlers with the query bus
func RegisterQueryHandlers(b cqrs.QueryBus, config *config.Config, ansible ansible.Repository, orchestrator orchestrator.Repository) error {
	if err := b.Register(get_app.NewGetAppQueryHandler(config.GetAnsibleAppsRolesPath(), config.GetAnsibleAppRoleCurrentVersionFolder())); err != nil {
		return log.Errorf("failed to register get app query handler: %v", err)
	}

	if err := b.Register(get_apps_status.NewGetAppsStatusQueryHandler(orchestrator)); err != nil {
		return log.Errorf("failed to register get apps status query handler: %v", err)
	}

	return nil
}
