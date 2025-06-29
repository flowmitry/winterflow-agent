package command

import (
	"winterflow-agent/internal/application/command/control_app"
	"winterflow-agent/internal/application/command/create_registry"
	"winterflow-agent/internal/application/command/delete_app"
	"winterflow-agent/internal/application/command/delete_registry"
	"winterflow-agent/internal/application/command/rename_app"
	"winterflow-agent/internal/application/command/save_app"
	"winterflow-agent/internal/application/command/update_agent"
	"winterflow-agent/internal/application/config"
	"winterflow-agent/internal/domain/repository"
	"winterflow-agent/internal/domain/service/app"
	"winterflow-agent/pkg/cqrs"
	"winterflow-agent/pkg/log"
)

func RegisterCommandHandlers(b cqrs.CommandBus, config *config.Config, appRepository repository.AppRepository, registryRepository repository.DockerRegistryRepository) error {
	versionService := app.NewAppVersionService(config)

	if err := b.Register(save_app.NewSaveAppHandler(config.GetAppsTemplatesPath(), config.GetPrivateKeyPath(), versionService)); err != nil {
		return log.Errorf("failed to register save app handler", "error", err)
	}

	if err := b.Register(delete_app.NewDeleteAppHandler(appRepository, config.GetAppsTemplatesPath())); err != nil {
		return log.Errorf("failed to register delete app handler", "error", err)
	}

	if err := b.Register(control_app.NewControlAppHandler(appRepository, versionService)); err != nil {
		return log.Errorf("failed to register control app handler", "error", err)
	}

	if err := b.Register(update_agent.NewUpdateAgentHandler(config)); err != nil {
		return log.Errorf("failed to register update agent handler", "error", err)
	}

	if err := b.Register(rename_app.NewRenameAppHandler(appRepository, config.GetAppsTemplatesPath(), versionService)); err != nil {
		return log.Errorf("failed to register rename app handler", "error", err)
	}

	if err := b.Register(create_registry.NewCreateRegistryHandler(registryRepository, config)); err != nil {
		return log.Errorf("failed to register create registry handler", "error", err)
	}

	if err := b.Register(delete_registry.NewDeleteRegistryHandler(registryRepository, config)); err != nil {
		return log.Errorf("failed to register delete registry handler", "error", err)
	}

	return nil
}
