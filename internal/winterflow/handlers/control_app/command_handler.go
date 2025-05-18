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
	log.Debug("Processing control app request for app ID: %s, action: %s", cmd.Request.AppId, cmd.Request.Action.String())

	// Validate the app ID
	appID := cmd.Request.AppId
	if appID == "" {
		return log.Errorf("app ID is required")
	}

	// Get the app config
	appConfig, err := GetAppConfig(appID)
	if err != nil {
		return log.Errorf("error getting app config: %w", err)
	}

	// Determine the action to perform
	var action string
	switch cmd.Request.Action {
	case pb.AppAction_START:
		action = "start"
	case pb.AppAction_STOP:
		action = "stop"
	case pb.AppAction_RESTART:
		action = "restart"
	default:
		return log.Errorf("unsupported action: %s", cmd.Request.Action.String())
	}

	// Construct the Ansible command
	// Use the app's role directly with the action as an extra variable
	command := []string{
		"-i", "localhost,",
		"-e", fmt.Sprintf("app_id=%s", appID),
		"-e", fmt.Sprintf("action=%s", action),
		fmt.Sprintf("playbooks/app/control.yml"),
	}

	result := h.ansible.RunSync(cmd.Request.Base.MessageId, command)
	if result.ExitCode != 0 {
		return log.Errorf("command failed with exit code %d: %s", result.ExitCode, result.Error)
	}

	log.Info("Successfully executed %s action on app %s", action, appConfig.Name)
	return nil
}

// NewControlAppHandler creates a new ControlAppHandler
func NewControlAppHandler(client *ansiblepkg.Client) *ControlAppHandler {
	return &ControlAppHandler{
		ansible: *client,
	}
}
