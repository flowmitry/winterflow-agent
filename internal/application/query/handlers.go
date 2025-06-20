package query

import (
	"winterflow-agent/internal/application/query/get_app"
	"winterflow-agent/internal/application/query/get_apps_status"
	"winterflow-agent/internal/config"
	"winterflow-agent/internal/domain/repository"
	"winterflow-agent/pkg/cqrs"
	"winterflow-agent/pkg/log"
)

func RegisterQueryHandlers(b cqrs.QueryBus, config *config.Config, ansible repository.RunnerRepository, containerAppRepository repository.ContainerAppRepository) error {
	if err := b.Register(get_app.NewGetAppQueryHandler(config.GetAnsibleAppsRolesPath(), config.GetAnsibleAppRoleCurrentVersionFolder())); err != nil {
		return log.Errorf("failed to register get app query handler: %v", err)
	}

	if err := b.Register(get_apps_status.NewGetAppsStatusQueryHandler(containerAppRepository)); err != nil {
		return log.Errorf("failed to register get apps status query handler: %v", err)
	}

	return nil
}
