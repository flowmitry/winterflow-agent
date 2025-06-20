package command

import (
	"winterflow-agent/internal/application/command/control_app"
	"winterflow-agent/internal/application/command/delete_app"
	"winterflow-agent/internal/application/command/save_app"
	"winterflow-agent/internal/application/command/update_agent"
	"winterflow-agent/internal/config"
	"winterflow-agent/internal/domain/repository"
	"winterflow-agent/pkg/cqrs"
	"winterflow-agent/pkg/log"
)

func RegisterCommandHandlers(b cqrs.CommandBus, config *config.Config, ansible repository.RunnerRepository) error {
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
