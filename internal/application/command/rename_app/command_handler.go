package rename_app

import (
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

	// Create a new revision
	newVersion, err := h.VersionService.CreateRevision(appID)
	if err != nil {
		return log.Errorf("failed to create new revision for app", "app_id", appID, "error", err)
	}
	log.Debug("Created new revision for app", "app_id", appID, "new_revision", newVersion)

	// Re-Deploy the new revision
	if err := h.repository.RenameApp(appID, newName); err != nil {
		return log.Errorf("repository rename failed", "error", err)
	}

	log.Info("Successfully renamed app", "app_id", appID, "new_name", newName, "new_revision", newVersion)
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
