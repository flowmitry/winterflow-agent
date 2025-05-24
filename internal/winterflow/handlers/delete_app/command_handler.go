package delete_app

import (
	"os"
	"path/filepath"
	log "winterflow-agent/pkg/log"
)

// DeleteAppHandler handles the DeleteAppCommand
type DeleteAppHandler struct {
	AnsibleAppsRolesPath string
}

// Handle executes the DeleteAppCommand
func (h *DeleteAppHandler) Handle(cmd DeleteAppCommand) error {
	if cmd.Request == nil {
		return log.Errorf("invalid request: request is nil")
	}

	if cmd.Request.Base == nil {
		return log.Errorf("invalid request: base message is nil")
	}

	messageID := cmd.Request.Base.MessageId
	appID := cmd.Request.AppId
	log.Debug("Processing delete app request for message ID: %s, app ID: %s", messageID, appID)

	// Validate the app ID
	if appID == "" {
		return log.Errorf("app ID is required for delete app command")
	}

	// Check if the app exists
	appDir := filepath.Join(h.AnsibleAppsRolesPath, appID)
	if _, err := os.Stat(appDir); os.IsNotExist(err) {
		return log.Errorf("app with ID %s does not exist", appID)
	}

	// Delete the app directory
	if err := os.RemoveAll(appDir); err != nil {
		return log.Errorf("failed to delete app directory for app ID %s: %w", appID, err)
	}

	log.Info("Successfully deleted app with ID: %s", appID)
	return nil
}

// NewDeleteAppHandler creates a new DeleteAppHandler
func NewDeleteAppHandler(ansibleAppsRolesPath string) *DeleteAppHandler {
	return &DeleteAppHandler{
		AnsibleAppsRolesPath: ansibleAppsRolesPath,
	}
}
