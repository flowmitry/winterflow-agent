package delete_app

import (
	"os"
	"path/filepath"
	"winterflow-agent/internal/winterflow/handlers/utils"
	ansiblepkg "winterflow-agent/pkg/ansible"
	log "winterflow-agent/pkg/log"
)

// DeleteAppHandler handles the DeleteAppCommand
type DeleteAppHandler struct {
	ansible                        ansiblepkg.Client
	AnsibleAppsRolesPath           string
	AnsibleAppsRolesCurrentVersion string
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
		log.Warn("App directory for app ID %s does not exist, it may have been already deleted", appID)
		log.Info("Successfully deleted app with ID: %s", appID)
		return nil
	}

	// Get the app config
	appConfig, err := utils.GetAppConfig(h.AnsibleAppsRolesPath, appID, h.AnsibleAppsRolesCurrentVersion)
	if err != nil {
		return log.Errorf("failed to get app config for app ID %s: %w", appID, err)
	}

	// Build environment variables
	env := map[string]string{
		"app_id":       appID,
		"orchestrator": appConfig.Type.String(),
	}
	ansibleCommand := ansiblepkg.Command{
		Id:       cmd.Request.Base.MessageId,
		Playbook: "apps/delete_app.yml",
		Env:      env,
	}

	log.Info("Executing delete_app playbook for app %s (ID: %s)", appConfig.Name, appID)

	result := h.ansible.RunSync(ansibleCommand)
	if result.ExitCode != 0 {
		return log.Errorf("command failed with exit code %d: %v", result.ExitCode, result.Error)
	}

	// Delete the app directory
	if err := os.RemoveAll(appDir); err != nil {
		return log.Errorf("failed to delete app directory for app ID %s: %w", appID, err)
	}

	log.Info("Successfully deleted app with ID: %s", appID)
	return nil
}

// NewDeleteAppHandler creates a new DeleteAppHandler
func NewDeleteAppHandler(client *ansiblepkg.Client, ansibleAppsRolesPath, ansibleAppsRolesCurrentVersion string) *DeleteAppHandler {
	return &DeleteAppHandler{
		ansible:                        *client,
		AnsibleAppsRolesPath:           ansibleAppsRolesPath,
		AnsibleAppsRolesCurrentVersion: ansibleAppsRolesCurrentVersion,
	}
}
