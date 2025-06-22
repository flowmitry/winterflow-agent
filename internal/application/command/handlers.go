package command

import (
	"winterflow-agent/internal/application/command/control_app"
	"winterflow-agent/internal/application/command/delete_app"
	"winterflow-agent/internal/application/command/rename_app"
	"winterflow-agent/internal/application/command/save_app"
	"winterflow-agent/internal/application/command/update_agent"
	"winterflow-agent/internal/application/config"
	"winterflow-agent/internal/domain/repository"
	"winterflow-agent/pkg/cqrs"
	"winterflow-agent/pkg/log"
)

func RegisterCommandHandlers(b cqrs.CommandBus, config *config.Config, appRepository repository.AppRepository) error {
	if err := b.Register(save_app.NewSaveAppHandler(config.GetAppsTemplatesPath(), config.GetAppsCurrentVersionFolder(), config.GetPrivateKeyPath())); err != nil {
		return log.Errorf("failed to register save app handler: %v", err)
	}

	if err := b.Register(delete_app.NewDeleteAppHandler(appRepository, config.GetAppsTemplatesPath(), config.GetAppsCurrentVersionFolder())); err != nil {
		return log.Errorf("failed to register delete app handler: %v", err)
	}

	if err := b.Register(control_app.NewControlAppHandler(appRepository, config.GetAppsTemplatesPath(), config.GetAppsCurrentVersionFolder())); err != nil {
		return log.Errorf("failed to register control app handler: %v", err)
	}

	if err := b.Register(update_agent.NewUpdateAgentHandler(config)); err != nil {
		return log.Errorf("failed to register update agent handler: %v", err)
	}

	if err := b.Register(rename_app.NewRenameAppHandler(appRepository, config.GetAppsTemplatesPath(), config.GetAppsCurrentVersionFolder())); err != nil {
		return log.Errorf("failed to register rename app handler: %v", err)
	}

	return nil
}
