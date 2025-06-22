package docker_compose

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"winterflow-agent/internal/domain/model"
)

// fileExists returns true if the provided path exists and is not a directory.
func fileExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !info.IsDir()
}

// getAppName reads the app's config.json located in the current-version folder
// and returns the `name` field. It falls back to appID if anything goes wrong.
func (r *composeRepository) getAppName(appID string) (string, error) {
	configPath := filepath.Join(
		r.config.GetAppsTemplatesPath(),
		appID,
		r.config.GetAppsCurrentVersionFolder(),
		"config.json",
	)

	data, err := os.ReadFile(configPath)
	if err != nil {
		return appID, fmt.Errorf("failed to read app config: %w", err)
	}

	appConfig, err := model.ParseAppConfig(data)
	if err != nil {
		return appID, fmt.Errorf("failed to parse app config: %w", err)
	}

	if strings.TrimSpace(appConfig.Name) == "" {
		return "", fmt.Errorf("application name is empty in config for app %s", appID)
	}
	return appConfig.Name, nil
}
