package control_app

import (
	"fmt"
	"winterflow-agent/internal/winterflow/ansible"
	"winterflow-agent/internal/winterflow/grpc/pb"
	"winterflow-agent/internal/winterflow/handlers/utils"
	ansiblepkg "winterflow-agent/pkg/ansible"
	log "winterflow-agent/pkg/log"
)

// ControlAppHandler handles the ControlAppCommand
type ControlAppHandler struct {
	ansible                        ansible.Repository
	AnsibleAppsRolesPath           string
	AnsibleAppsRolesCurrentVersion string
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
	appConfig, err := utils.GetAppConfig(h.AnsibleAppsRolesPath, appID, h.AnsibleAppsRolesCurrentVersion)
	if err != nil {
		return log.Errorf("failed to get app config for app ID %s: %w", appID, err)
	}

	// Determine the app version to use
	var appVersion string
	if cmd.Request.AppVersion > 0 {
		appVersion = fmt.Sprintf("%d", cmd.Request.AppVersion)
	} else {
		appVersion = h.AnsibleAppsRolesCurrentVersion
	}

	// Determine the action to perform
	var playbook string
	var result ansiblepkg.Result
	switch cmd.Request.Action {
	case pb.AppAction_START:
		playbook = "deploy_app"
		result = h.ansible.DeployApp(appID, appVersion)
	case pb.AppAction_STOP:
		playbook = "stop_app"
		result = h.ansible.StopApp(appID)
	case pb.AppAction_RESTART:
		playbook = "restart_app"
		result = h.ansible.RestartApp(appID, appVersion)
	case pb.AppAction_UPDATE:
		playbook = "update_app"
		result = h.ansible.UpdateApp(appID)
	default:
		return log.Errorf("unsupported action: %s", cmd.Request.Action.String())
	}

	if result.ExitCode != 0 {
		return log.Errorf("command failed with exit code %d: %v", result.ExitCode, result.Error)
	}

	log.Printf("Successfully executed %s playbook on app %s", playbook, appConfig.Name)
	return nil
}

// NewControlAppHandler creates a new ControlAppHandler
func NewControlAppHandler(ansible ansible.Repository, ansibleAppsRolesPath, ansibleAppsRolesCurrentVersion string) *ControlAppHandler {
	return &ControlAppHandler{
		ansible:                        ansible,
		AnsibleAppsRolesPath:           ansibleAppsRolesPath,
		AnsibleAppsRolesCurrentVersion: ansibleAppsRolesCurrentVersion,
	}
}
