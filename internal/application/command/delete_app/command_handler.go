package delete_app

import (
	"os"
	"path/filepath"
	"winterflow-agent/internal/domain/repository"
	log "winterflow-agent/pkg/log"
)

// DeleteAppHandler handles the DeleteAppCommand
type DeleteAppHandler struct {
	ansible                        repository.RunnerRepository
	AnsibleAppsRolesPath           string
	AnsibleAppsRolesCurrentVersion string
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
	appDir := filepath.Join(h.AnsibleAppsRolesPath, appID)
	if _, err := os.Stat(appDir); os.IsNotExist(err) {
		log.Warn("App directory for app ID %s does not exist, it may have been already deleted", appID)
		log.Info("Successfully deleted app with ID: %s", appID)
		return nil
	}

	result := h.ansible.DeleteApp(appID)
	if result.ExitCode != 0 {
		return log.Errorf("Deletion app command failed with exit code %d: %v", result.ExitCode, result.Error)
	}

	// Delete the app directory
	if err := os.RemoveAll(appDir); err != nil {
		return log.Errorf("failed to delete app directory for app ID %s: %w", appID, err)
	}

	log.Info("Successfully deleted app with ID: %s", appID)
	return nil
}

// NewDeleteAppHandler creates a new DeleteAppHandler
func NewDeleteAppHandler(ansible repository.RunnerRepository, ansibleAppsRolesPath, ansibleAppsRolesCurrentVersion string) *DeleteAppHandler {
	return &DeleteAppHandler{
		ansible:                        ansible,
		AnsibleAppsRolesPath:           ansibleAppsRolesPath,
		AnsibleAppsRolesCurrentVersion: ansibleAppsRolesCurrentVersion,
	}
}
