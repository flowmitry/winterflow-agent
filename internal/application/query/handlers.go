package query

import (
	"winterflow-agent/internal/application/config"
	"winterflow-agent/internal/application/query/get_app"
	"winterflow-agent/internal/application/query/get_apps_status"
	"winterflow-agent/internal/domain/repository"
	appservice "winterflow-agent/internal/domain/service/app"
	"winterflow-agent/pkg/cqrs"
	"winterflow-agent/pkg/log"
)

func RegisterQueryHandlers(b cqrs.QueryBus, config *config.Config, appRepository repository.AppRepository) error {
	// Initialise the service responsible for application versions.
	versionService := appservice.NewAppVersionService(config)

	if err := b.Register(get_app.NewGetAppQueryHandler(versionService)); err != nil {
		return log.Errorf("failed to register get app query handler: %v", err)
	}

	if err := b.Register(get_apps_status.NewGetAppsStatusQueryHandler(appRepository)); err != nil {
		return log.Errorf("failed to register get apps status query handler: %v", err)
	}

	return nil
}
