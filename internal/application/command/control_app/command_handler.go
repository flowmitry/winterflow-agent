package control_app

import (
	"fmt"
	"os"
	"path/filepath"
	"winterflow-agent/internal/domain/model"
	"winterflow-agent/internal/domain/repository"
	log "winterflow-agent/pkg/log"
)

// ControlAppHandler handles the ControlAppCommand
type ControlAppHandler struct {
	repository         repository.AppRepository
	AppsTemplatesPath  string
	AppsCurrentVersion string
}

// Handle executes the ControlAppCommand
func (h *ControlAppHandler) Handle(cmd ControlAppCommand) error {
	log.Debug("Processing control app request for app ID: %s, action: %d", cmd.AppID, cmd.Action)

	// Validate the app ID
	if cmd.AppID == "" {
		return log.Errorf("app ID is required for control app command")
	}

	// Get the app config
	appConfig, err := getAppConfig(h.AppsTemplatesPath, cmd.AppID, h.AppsCurrentVersion)
	if err != nil {
		return log.Errorf("failed to get app config for app ID %s: %w", cmd.AppID, err)
	}

	// Determine the app version to use
	var appVersion string
	if cmd.AppVersion > 0 {
		appVersion = fmt.Sprintf("%d", cmd.AppVersion)
	} else {
		appVersion = h.AppsCurrentVersion
	}

	// Determine the action to perform
	var playbook string
	var actionErr error
	switch cmd.Action {
	case AppActionStart:
		playbook = "deploy_app"
		actionErr = h.repository.DeployApp(cmd.AppID, appVersion)
	case AppActionStop:
		playbook = "stop_app"
		actionErr = h.repository.StopApp(cmd.AppID)
	case AppActionRestart:
		playbook = "restart_app"
		actionErr = h.repository.RestartApp(cmd.AppID, appVersion)
	case AppActionUpdate:
		playbook = "update_app"
		actionErr = h.repository.UpdateApp(cmd.AppID)
	default:
		return log.Errorf("unsupported action: %d", cmd.Action)
	}

	if actionErr != nil {
		return log.Errorf("command failed with error: %v", actionErr)
	}

	log.Printf("Successfully executed %s playbook on app %s", playbook, appConfig.Name)
	return nil
}

// getAppConfig retrieves the app configuration for the given app ID
func getAppConfig(appsTemplatesPath, appID, version string) (*model.AppConfig, error) {
	// Check if the app exists
	appDir := filepath.Join(appsTemplatesPath, appID, version)
	if _, err := os.Stat(appDir); os.IsNotExist(err) {
		return nil, fmt.Errorf("app with ID %s does not exist", appID)
	}

	// Get the app config
	configFile := filepath.Join(appDir, "config.json")
	configBytes, err := os.ReadFile(configFile)
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
func NewControlAppHandler(repository repository.AppRepository, appsTemplatesPath, appsCurrentVersion string) *ControlAppHandler {
	return &ControlAppHandler{
		repository:         repository,
		AppsTemplatesPath:  appsTemplatesPath,
		AppsCurrentVersion: appsCurrentVersion,
	}
}
