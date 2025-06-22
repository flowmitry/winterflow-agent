package rename_app

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"winterflow-agent/internal/domain/model"
	"winterflow-agent/internal/domain/repository"
	log "winterflow-agent/pkg/log"
)

// RenameAppHandler handles the RenameAppCommand.
type RenameAppHandler struct {
	repository         repository.AppRepository
	AppsTemplatesPath  string
	AppsCurrentVersion string
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
		return log.Errorf("application name '%s' is already in use by another app", newName)
	}

	// Path to config.json of the current version.
	configPath := filepath.Join(h.AppsTemplatesPath, appID, h.AppsCurrentVersion, "config.json")

	// Read existing config.
	cfgBytes, err := os.ReadFile(configPath)
	if err != nil {
		return log.Errorf("failed to read app config: %w", err)
	}

	cfg, err := model.ParseAppConfig(cfgBytes)
	if err != nil {
		return log.Errorf("failed to parse app config: %w", err)
	}

	// If the name is unchanged, we only need to update container directory if necessary.
	if strings.EqualFold(strings.TrimSpace(cfg.Name), newName) {
		log.Info("[RenameApp] Name is unchanged – skipping config update", "app_id", appID)
	} else {
		cfg.Name = newName
		data, err := json.Marshal(cfg)
		if err != nil {
			return log.Errorf("failed to marshal updated app config: %w", err)
		}
		if err := os.WriteFile(configPath, data, 0o644); err != nil {
			return log.Errorf("failed to write updated app config: %w", err)
		}
		log.Debug("Updated config.json with new application name", "path", configPath)
	}

	// Trigger repository-level rename (e.g., docker-compose project directory).
	if err := h.repository.RenameApp(appID, newName); err != nil {
		return log.Errorf("repository rename failed: %w", err)
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

		cfgPath := filepath.Join(h.AppsTemplatesPath, appID, h.AppsCurrentVersion, "config.json")
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
func NewRenameAppHandler(repository repository.AppRepository, appsTemplatesPath, appsCurrentVersion string) *RenameAppHandler {
	return &RenameAppHandler{
		repository:         repository,
		AppsTemplatesPath:  appsTemplatesPath,
		AppsCurrentVersion: appsCurrentVersion,
	}
}
