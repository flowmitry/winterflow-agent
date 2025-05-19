package control_app

import (
	"fmt"
	"winterflow-agent/internal/winterflow/grpc/pb"
	ansiblepkg "winterflow-agent/pkg/ansible"
	log "winterflow-agent/pkg/log"
)

// ControlAppHandler handles the ControlAppCommand
type ControlAppHandler struct {
	ansible ansiblepkg.Client
}

// Handle executes the ControlAppCommand
func (h *ControlAppHandler) Handle(cmd ControlAppCommand) error {
	if cmd.Request == nil {
		return log.Errorf("invalid request: request is nil")
	}

	if cmd.Request.Base == nil {
		return log.Errorf("invalid request: base message is nil")
	}

	messageID := cmd.Request.Base.MessageId
	log.Debug("Processing control app request for message ID: %s, app ID: %s, action: %s",
		messageID, cmd.Request.AppId, cmd.Request.Action.String())

	// Validate the app ID
	appID := cmd.Request.AppId
	if appID == "" {
		return log.Errorf("app ID is required for control app command")
	}

	// Get the app config
	appConfig, err := GetAppConfig(appID)
	if err != nil {
		return log.Errorf("failed to get app config for app ID %s: %w", appID, err)
	}

	// Determine the action to perform
	var playbook string
	switch cmd.Request.Action {
	case pb.AppAction_START:
		playbook = "deploy_app"
	case pb.AppAction_STOP:
		playbook = "stop_app"
	case pb.AppAction_RESTART:
		playbook = "restart_app"
	default:
		return log.Errorf("unsupported action: %s", cmd.Request.Action.String())
	}

	// Determine the app version to use
	appVersion := "latest" // Default version

	// Build environment variables
	env := map[string]string{
		"app_id":      fmt.Sprintf("app_id=%s", appID),
		"app_version": fmt.Sprintf("app_version=%s", appVersion),
		"app_name":    fmt.Sprintf("app_name=%s", appConfig.Name),
	}

	// Add any additional environment variables from app configuration if needed

	ansibleCommand := ansiblepkg.Command{
		Playbook: fmt.Sprintf("apps/%s.yml", playbook),
		Env:      env,
	}

	log.Info("Executing %s playbook for app %s (ID: %s, Version: %s)",
		playbook, appConfig.Name, appID, appVersion)

	result := h.ansible.RunSync(cmd.Request.Base.MessageId, ansibleCommand)
	if result.ExitCode != 0 {
		return log.Errorf("command failed with exit code %d: %v", result.ExitCode, result.Error)
	}

	log.Info("Successfully executed %s playbook on app %s", playbook, appConfig.Name)
	return nil
}

// NewControlAppHandler creates a new ControlAppHandler
func NewControlAppHandler(client *ansiblepkg.Client) *ControlAppHandler {
	return &ControlAppHandler{
		ansible: *client,
	}
}
