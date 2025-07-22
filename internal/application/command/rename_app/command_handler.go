package rename_app

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"winterflow-agent/internal/domain/model"
	"winterflow-agent/internal/domain/repository"
	"winterflow-agent/internal/domain/service/app"
	"winterflow-agent/pkg/log"
)

// RenameAppHandler handles the RenameAppCommand.
type RenameAppHandler struct {
	repository        repository.AppRepository
	AppsTemplatesPath string
	VersionService    app.RevisionServiceInterface
}

// Handle executes the RenameAppCommand.
func (h *RenameAppHandler) Handle(cmd RenameAppCommand) error {
	appID := strings.TrimSpace(cmd.AppID)
	newName := strings.TrimSpace(cmd.AppName)

	log.Debug("Processing rename app request", "app_id", appID, "new_name", newName)

	if appID == "" {
		return log.Errorf("app ID is required for rename app command")
	}
	if newName == "" {
		return log.Errorf("new app name cannot be empty")
	}

	// Ensure uniqueness of the new name.
	unique, err := h.isNameUnique(newName, appID)
	if err != nil {
		return err
	}
	if !unique {
		return log.Errorf("application name is already in use by another app", "app_name", newName)
	}

	// Resolve the latest version directory via the version service.
	latestVersion, err := h.VersionService.GetLatestAppRevision(appID)
	if err != nil {
		return log.Errorf("failed to determine latest version for app", "app_id", appID, "error", err)
	}
	if latestVersion == 0 {
		return log.Errorf("application does not have any versions yet", "app_id", appID)
	}

	configPath := filepath.Join(h.VersionService.GetRevisionDir(appID, latestVersion), "config.json")

	// Read existing config.
	cfgBytes, err := os.ReadFile(configPath)
	if err != nil {
		return log.Errorf("failed to read app config", "error", err)
	}

	cfg, err := model.ParseAppConfig(cfgBytes)
	if err != nil {
		return log.Errorf("failed to parse app config", "error", err)
	}

	// First, rename the container directory via the repository. This must happen BEFORE we
	// change the config.json so that repository.getAppName() still returns the old name.
	if err := h.repository.RenameApp(appID, newName); err != nil {
		return log.Errorf("repository rename failed", "error", err)
	}

	// Now update config.json with the new name (if it actually changed).
	if !strings.EqualFold(strings.TrimSpace(cfg.Name), newName) {
		cfg.Name = newName
		data, err := json.Marshal(cfg)
		if err != nil {
			return log.Errorf("failed to marshal updated app config", "error", err)
		}
		if err := os.WriteFile(configPath, data, 0o644); err != nil {
			return log.Errorf("failed to write updated app config", "error", err)
		}
		log.Debug("Updated config.json with new application name", "path", configPath)
	} else {
		log.Info("[RenameApp] Name is unchanged – skipping config update", "app_id", appID)
	}

	log.Info("Successfully renamed app", "app_id", appID, "new_name", newName)
	return nil
}

// isNameUnique checks that the provided application name is not used by any other application (except the current one).
func (h *RenameAppHandler) isNameUnique(name, currentAppID string) (bool, error) {
	entries, err := os.ReadDir(h.AppsTemplatesPath)
	if err != nil {
		return false, fmt.Errorf("failed to read apps templates directory: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		appID := entry.Name()
		if appID == currentAppID {
			// Skip current application.
			continue
		}

		// Determine latest version for each application to read its current name.
		latestVersion, err := h.VersionService.GetLatestAppRevision(appID)
		if err != nil || latestVersion == 0 {
			continue
		}

		cfgPath := filepath.Join(h.VersionService.GetRevisionDir(appID, latestVersion), "config.json")
		data, err := os.ReadFile(cfgPath)
		if err != nil {
			// Ignore missing or unreadable config files.
			continue
		}
		cfg, err := model.ParseAppConfig(data)
		if err != nil {
			continue
		}
		if strings.EqualFold(strings.TrimSpace(cfg.Name), strings.TrimSpace(name)) {
			return false, nil
		}
	}
	return true, nil
}

// NewRenameAppHandler creates a new RenameAppHandler.
func NewRenameAppHandler(repository repository.AppRepository, appsTemplatesPath string, versionService app.RevisionServiceInterface) *RenameAppHandler {
	return &RenameAppHandler{
		repository:        repository,
		AppsTemplatesPath: appsTemplatesPath,
		VersionService:    versionService,
	}
}
