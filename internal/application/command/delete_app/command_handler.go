package delete_app

import (
	"os"
	"path/filepath"
	"winterflow-agent/internal/domain/repository"
	log "winterflow-agent/pkg/log"
)

// DeleteAppHandler handles the DeleteAppCommand
type DeleteAppHandler struct {
	repository        repository.AppRepository
	AppsTemplatesPath string
}

// Handle executes the DeleteAppCommand
func (h *DeleteAppHandler) Handle(cmd DeleteAppCommand) error {
	appID := cmd.AppID
	log.Debug("Processing delete app request for app ID: %s", appID)

	// Validate the app ID
	if appID == "" {
		return log.Errorf("app ID is required for delete app command")
	}

	// Check if the app exists
	appDir := filepath.Join(h.AppsTemplatesPath, appID)
	if _, err := os.Stat(appDir); os.IsNotExist(err) {
		log.Warn("App directory for app ID %s does not exist, it may have been already deleted", appID)
		log.Info("Successfully deleted app with ID: %s", appID)
		return nil
	}

	err := h.repository.DeleteApp(appID)
	if err != nil {
		return log.Errorf("Deletion app command failed with error: %v", err)
	}

	// Delete the app directory
	if err := os.RemoveAll(appDir); err != nil {
		return log.Errorf("failed to delete app directory for app ID %s: %w", appID, err)
	}

	log.Info("Successfully deleted app with ID: %s", appID)
	return nil
}

// NewDeleteAppHandler creates a new DeleteAppHandler
func NewDeleteAppHandler(repository repository.AppRepository, appsTemplatesPath string) *DeleteAppHandler {
	return &DeleteAppHandler{
		repository:        repository,
		AppsTemplatesPath: appsTemplatesPath,
	}
}
