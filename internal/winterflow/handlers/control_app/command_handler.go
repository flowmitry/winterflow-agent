package control_app

import (
	"fmt"
	"winterflow-agent/internal/winterflow/grpc/pb"
	"winterflow-agent/internal/winterflow/handlers/utils"
	ansiblepkg "winterflow-agent/pkg/ansible"
	log "winterflow-agent/pkg/log"
)

// ControlAppHandler handles the ControlAppCommand
type ControlAppHandler struct {
	ansible              ansiblepkg.Client
	AnsibleAppsRolesPath string
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
	appConfig, err := utils.GetAppConfig(h.AnsibleAppsRolesPath, appID)
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
	var appVersion string
	if cmd.Request.AppVersion > 0 {
		appVersion = fmt.Sprintf("%d", cmd.Request.AppVersion)
	} else {
		appVersion = h.ansible.GetConfig().AnsibleAppsRolesCurrentVersion
	}

	// Build environment variables
	env := map[string]string{
		"app_id":         fmt.Sprintf("%s", appID),
		"app_version":    fmt.Sprintf("%s", appVersion),
		"apps_roles_dir": fmt.Sprintf("%s", h.AnsibleAppsRolesPath),
		"orchestrator":   fmt.Sprintf("%s", appConfig.Type),
	}
	ansibleCommand := ansiblepkg.Command{
		Id:       cmd.Request.Base.MessageId,
		Playbook: fmt.Sprintf("apps/%s.yml", playbook),
		Env:      env,
	}

	log.Info("Executing %s playbook for app %s (ID: %s, Version: %s)",
		playbook, appConfig.Name, appID, appVersion)

	result := h.ansible.RunSync(ansibleCommand)
	if result.ExitCode != 0 {
		return log.Errorf("command failed with exit code %d: %v", result.ExitCode, result.Error)
	}

	log.Info("Successfully executed %s playbook on app %s", playbook, appConfig.Name)
	return nil
}

// NewControlAppHandler creates a new ControlAppHandler
func NewControlAppHandler(client *ansiblepkg.Client, ansibleAppsRolesPath string) *ControlAppHandler {
	return &ControlAppHandler{
		ansible:              *client,
		AnsibleAppsRolesPath: ansibleAppsRolesPath,
	}
}
