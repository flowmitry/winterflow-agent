package docker_compose

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"winterflow-agent/internal/domain/model"
	appsvc "winterflow-agent/internal/domain/service/app"
)

// fileExists returns true if the provided path exists and is not a directory.
func fileExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !info.IsDir()
}

// dirExists returns true if the provided path exists and **is** a directory.
func dirExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.IsDir()
}

// ensureDir checks if the given directory exists and creates it (including
// parent directories) if it does not. It returns an error only if the
// directory does not exist and cannot be created.
func ensureDir(path string) error {
	if dirExists(path) {
		return nil
	}
	return os.MkdirAll(path, 0o755) // Create with typical rwxr-xr-x permissions
}

// getAppName determines the human-readable application name by inspecting the
// config.json stored in the latest version directory.
//
// The function falls back to returning appID when it cannot reliably detect a
// name (e.g. missing config.json or empty name field). This preserves previous
// behaviour where the app ID served as a safe default.
func (r *composeRepository) getAppNameById(appID string) (string, error) {
	// Instantiate the version service on-demand – the constructor is cheap and
	// avoids having to store another dependency inside the repository struct.
	versionService := appsvc.NewAppVersionService(r.config)

	// Always work with the latest version of the application. If no version
	// exists yet we fall back to 1 (legacy default) to keep compatibility with
	// older deployments that did not support versioning.
	latest, err := versionService.GetLatestAppVersion(appID)
	if err != nil {
		return appID, fmt.Errorf("failed to determine latest version for app %s: %w", appID, err)
	}

	if latest == 0 {
		return appID, fmt.Errorf("application %s has no versions", appID)
	}
	version := latest

	return getAppName(versionService.GetVersionDir(appID, version))
}

func (r *composeRepository) getAppName(appPath string) (string, error) {
	return getAppName(appPath)
}

// getAppName reads the application configuration located at the provided path
func getAppName(appPath string) (string, error) {
	if strings.TrimSpace(appPath) == "" {
		return "", fmt.Errorf("app path cannot be empty")
	}

	// Construct path to config.json relative to the provided directory.
	configPath := filepath.Join(appPath, "config.json")

	data, err := os.ReadFile(configPath)
	if err != nil {
		return "", fmt.Errorf("failed to read app config: %w", err)
	}

	appConfig, err := model.ParseAppConfig(data)
	if err != nil {
		return "", fmt.Errorf("failed to parse app config: %w", err)
	}

	name := strings.TrimSpace(appConfig.Name)
	if name == "" {
		return "", fmt.Errorf("application name is empty in config %s", configPath)
	}

	return name, nil
}
