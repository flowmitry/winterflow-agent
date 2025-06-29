package control_app

import (
	"fmt"
	"os"
	"path/filepath"
	"winterflow-agent/internal/domain/model"
	"winterflow-agent/internal/domain/repository"
	"winterflow-agent/internal/domain/service/app"
	"winterflow-agent/pkg/log"
)

// ControlAppHandler handles the ControlAppCommand
type ControlAppHandler struct {
	repository     repository.AppRepository
	VersionService app.AppVersionServiceInterface
}

// Handle executes the ControlAppCommand
func (h *ControlAppHandler) Handle(cmd ControlAppCommand) error {
	log.Debug("Processing control app request", "app_id", cmd.AppID, "action", cmd.Action)

	// Validate the app ID
	if cmd.AppID == "" {
		return log.Errorf("app ID is required for control app command")
	}

	// Determine the target version for the operation.
	var targetVersion uint32
	if cmd.AppVersion > 0 {
		// Validate requested version exists.
		exists, err := h.VersionService.ValidateAppVersion(cmd.AppID, cmd.AppVersion)
		if err != nil {
			return log.Errorf("failed to validate app version: %w", err)
		}
		if !exists {
			return log.Errorf("version %d not found for app %s", cmd.AppVersion, cmd.AppID)
		}
		targetVersion = cmd.AppVersion
	} else {
		latest, err := h.VersionService.GetLatestAppVersion(cmd.AppID)
		if err != nil {
			return log.Errorf("failed to determine latest version for app %s: %w", cmd.AppID, err)
		}
		if latest == 0 {
			return log.Errorf("no versions found for app %s", cmd.AppID)
		}
		targetVersion = latest
	}

	// Load app config for logging purposes.
	appConfig, err := getAppConfig(h.VersionService, cmd.AppID, targetVersion)
	if err != nil {
		return log.Errorf("failed to load app config: %w", err)
	}

	// Determine the action to perform
	var playbook string
	var actionErr error
	switch cmd.Action {
	case AppActionStart:
		playbook = "deploy_app"
		actionErr = h.repository.DeployApp(cmd.AppID)
	case AppActionStop:
		playbook = "stop_app"
		actionErr = h.repository.StopApp(cmd.AppID)
	case AppActionRestart:
		playbook = "restart_app"
		actionErr = h.repository.RestartApp(cmd.AppID)
	case AppActionUpdate:
		playbook = "update_app"
		actionErr = h.repository.UpdateApp(cmd.AppID)
	case AppActionRedeploy:
		playbook = "redeploy_app"
		actionErr = h.repository.StopApp(cmd.AppID)
		if actionErr != nil {
			return log.Errorf("command failed with error: %v", actionErr)
		}
		actionErr = h.repository.DeployApp(cmd.AppID)
	default:
		return log.Errorf("unsupported action: %d", cmd.Action)
	}

	if actionErr != nil {
		return log.Errorf("command failed with error: %v", actionErr)
	}

	log.Info("Successfully executed playbook", "playbook", playbook, "app_name", appConfig.Name)
	return nil
}

// getAppConfig retrieves the app configuration for the given app ID
func getAppConfig(versionService app.AppVersionServiceInterface, appID string, version uint32) (*model.AppConfig, error) {
	// Determine directory for the specified version.
	versionDir := versionService.GetVersionDir(appID, version)

	// Ensure it exists.
	if _, err := os.Stat(versionDir); os.IsNotExist(err) {
		return nil, fmt.Errorf("app with ID %s and version %d does not exist", appID, version)
	}

	// Read config.json
	configPath := filepath.Join(versionDir, "config.json")
	configBytes, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("error reading app config: %w", err)
	}

	appConfig, err := model.ParseAppConfig(configBytes)
	if err != nil {
		return nil, fmt.Errorf("error parsing app config: %w", err)
	}

	return appConfig, nil
}

// NewControlAppHandler creates a new ControlAppHandler
func NewControlAppHandler(repository repository.AppRepository, versionService app.AppVersionServiceInterface) *ControlAppHandler {
	return &ControlAppHandler{
		repository:     repository,
		VersionService: versionService,
	}
}
