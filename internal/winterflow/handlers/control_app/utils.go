package control_app

import (
	"fmt"
	"os"
	"path/filepath"
	"winterflow-agent/internal/config"
	"winterflow-agent/internal/winterflow/models"
)

// GetAppConfig retrieves the app configuration for the given app ID
func GetAppConfig(appID string) (*models.AppConfig, error) {
	// Check if the app exists
	appDir := filepath.Join(config.GetAnsibleAppsRolesPath(), appID)
	if _, err := os.Stat(appDir); os.IsNotExist(err) {
		return nil, fmt.Errorf("app with ID %s does not exist", appID)
	}

	// Get the app config
	configFile := filepath.Join(appDir, "config.json")
	configBytes, err := os.ReadFile(configFile)
	if err != nil {
		return nil, fmt.Errorf("error reading app config: %w", err)
	}

	appConfig, err := models.ParseAppConfig(configBytes)
	if err != nil {
		return nil, fmt.Errorf("error parsing app config: %w", err)
	}

	return appConfig, nil
}
