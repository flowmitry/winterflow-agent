package get_app_logs

import (
	"fmt"
	"winterflow-agent/internal/application/config"
	"winterflow-agent/internal/domain/model"
	"winterflow-agent/internal/domain/repository"
	"winterflow-agent/pkg/log"
)

// GetAppLogsQueryHandler handles the GetAppLogsQuery.
type GetAppLogsQueryHandler struct {
	appRepository repository.AppRepository
	config        *config.Config
}

// Handle executes the GetAppLogsQuery and returns the logs.
func (h *GetAppLogsQueryHandler) Handle(query GetAppLogsQuery) (*model.Logs, error) {
	if h.appRepository == nil {
		return nil, fmt.Errorf("appRepository is not configured")
	}

	// Check if app logs feature is disabled
	if h.config != nil && !h.config.IsFeatureEnabled(config.FeatureAppLogs) {
		return nil, log.Errorf("logs operations are disabled by configuration")
	}

	log.Info("Processing get app logs request", "app_id", query.AppID, "tail", query.Tail)

	logs, err := h.appRepository.GetLogs(query.AppID, query.Since, query.Until, query.Tail)
	if err != nil {
		log.Error("Error getting app logs", "error", err)
		return nil, fmt.Errorf("failed to get app logs: %w", err)
	}

	return &logs, nil
}

// NewGetAppLogsQueryHandler creates a new GetAppLogsQueryHandler.
func NewGetAppLogsQueryHandler(appRepo repository.AppRepository, cfg *config.Config) *GetAppLogsQueryHandler {
	return &GetAppLogsQueryHandler{appRepository: appRepo, config: cfg}
}
