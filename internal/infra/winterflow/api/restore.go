package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"time"

	"github.com/google/uuid"

	"winterflow-agent/internal/application/config"
	domain "winterflow-agent/internal/domain/model"
	"winterflow-agent/pkg/certs"
	"winterflow-agent/pkg/log"
)

type ExtensionValue struct {
	Extension      string `json:"extension"`
	ExtensionAppID string `json:"extension_app_id"`
}

// AppInfo mirrors the server-side structure used by /api/v1/data/restore.
// It is duplicated locally to avoid importing server packages.
type AppInfo struct {
	ID              string                  `json:"id"`
	TemplateID      string                  `json:"template_id"`
	Version         string                  `json:"version"`
	Name            string                  `json:"name"`
	Icon            string                  `json:"icon"`
	Color           string                  `json:"color"`
	ExtensionValues []domain.ExtensionValue `json:"extension_values"`
}

// restoreDataRequest matches the payload expected by the backend.
type restoreDataRequest struct {
	AgentID   string    `json:"agent_id"`
	Timestamp string    `json:"timestamp"`
	Secret    string    `json:"secret"`
	Apps      []AppInfo `json:"apps"`
}

// RestoreAgentData scans the local apps_templates folder, regenerates UUIDs,
// keeps only the latest revision for every app, updates the config.json files
// accordingly and notifies the WinterFlow backend via /api/v1/data/restore.
//
// It is intended to be executed via `winterflow-agent --restore` after the
// agent has been re-installed on a server while preserving the data volume.
func RestoreAgentData(configPath string) error {
	log.Info("Starting restore procedure")

	// ---------------------------------------------------------------------
	// 1. Load and validate configuration
	// ---------------------------------------------------------------------
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}
	if cfg.AgentStatus != config.AgentStatusRegistered || cfg.AgentID == "" {
		return fmt.Errorf("agent must be registered before running --restore")
	}

	// ---------------------------------------------------------------------
	// 2. Create backup of apps_templates if it doesn't exist
	// ---------------------------------------------------------------------
	templatesRoot := cfg.GetAppsTemplatesPath()
	backupRoot := filepath.Join(cfg.BasePath, "apps_templates.bak")

	if _, err := os.Stat(backupRoot); err == nil {
		// directory exists
		return fmt.Errorf("backup directory already exists: %s – aborting to prevent overwrite", backupRoot)
	}

	log.Info("Creating backup of application templates", "source", templatesRoot, "destination", backupRoot)
	if err := copyDirectoryRecursive(templatesRoot, backupRoot); err != nil {
		return fmt.Errorf("failed to create backup: %w", err)
	}
	log.Info("Backup created successfully", "path", backupRoot)

	// ---------------------------------------------------------------------
	// 3. Iterate over apps_templates and rewrite structure
	// ---------------------------------------------------------------------
	entries, err := os.ReadDir(templatesRoot)
	if err != nil {
		return fmt.Errorf("cannot read apps_templates directory %s: %w", templatesRoot, err)
	}

	var apps []AppInfo

	// Map of original app IDs to newly generated IDs so we can later update
	// any cross-references in extension_values.extension_app_id.
	oldToNewIDs := make(map[string]string)

	// Keep track of paths we need to revisit for updating configs once the
	// full mapping is known.
	type processedApp struct {
		newAppPath      string
		newRevisionPath string
		oldID           string
	}
	var processedApps []processedApp

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		oldAppID := entry.Name()
		oldAppPath := filepath.Join(templatesRoot, oldAppID)

		// Determine latest revision subdirectory (highest numeric name).
		versions, err := os.ReadDir(oldAppPath)
		if err != nil {
			log.Error("Failed to list versions", "app", oldAppID, "error", err)
			continue
		}

		var versionNumbers []int
		versionDirNames := make(map[int]string)
		for _, v := range versions {
			if !v.IsDir() {
				continue
			}
			n, err := strconv.Atoi(v.Name())
			if err != nil {
				// Skip non-numeric directories silently.
				continue
			}
			versionNumbers = append(versionNumbers, n)
			versionDirNames[n] = v.Name()
		}
		if len(versionNumbers) == 0 {
			// Nothing to process.
			continue
		}
		sort.Ints(versionNumbers)
		latestVersion := versionNumbers[len(versionNumbers)-1]
		latestDirName := versionDirNames[latestVersion]

		// Generate new UUID for the app and remember the mapping.
		newAppID := uuid.New().String()
		oldToNewIDs[oldAppID] = newAppID

		newAppPath := filepath.Join(templatesRoot, newAppID)
		newRevisionPath := filepath.Join(newAppPath, "1")

		// Record for the second processing phase.
		processedApps = append(processedApps, processedApp{
			newAppPath:      newAppPath,
			newRevisionPath: newRevisionPath,
			oldID:           oldAppID,
		})

		// Make sure parent directory exists.
		if err := os.MkdirAll(newAppPath, 0755); err != nil {
			log.Error("Failed to create new app directory", "path", newAppPath, "error", err)
			continue
		}

		// Move (rename) latest version directory to the new location.
		src := filepath.Join(oldAppPath, latestDirName)
		if err := os.Rename(src, newRevisionPath); err != nil {
			log.Error("Failed to move version directory", "src", src, "dst", newRevisionPath, "error", err)
			continue
		}

		// Before deleting the original directory, preserve current.config.json if present.
		oldCurrentCfgPath := filepath.Join(oldAppPath, "current.config.json")
		var currentCfgBytes []byte
		if data, err := os.ReadFile(oldCurrentCfgPath); err == nil {
			currentCfgBytes = data
		}

		_ = os.RemoveAll(oldAppPath)

		// -----------------------------------------------------------------
		// 2.1 Update config.json with new app ID
		// -----------------------------------------------------------------
		cfgPath := filepath.Join(newRevisionPath, "config.json")
		cfgBytes, err := os.ReadFile(cfgPath)
		if err != nil {
			log.Error("Failed to read config.json", "path", cfgPath, "error", err)
			continue
		}

		appCfg, err := domain.ParseAppConfig(cfgBytes)
		if err != nil {
			log.Error("Failed to parse app config", "path", cfgPath, "error", err)
			continue
		}

		appCfg.ID = newAppID

		newCfgBytes, err := json.MarshalIndent(appCfg, "", "  ")
		if err != nil {
			log.Error("Failed to marshal updated app config", "app", newAppID, "error", err)
			continue
		}

		if err := os.WriteFile(cfgPath, newCfgBytes, 0644); err != nil {
			log.Error("Failed to write updated config.json", "path", cfgPath, "error", err)
			continue
		}

		// -----------------------------------------------------------------
		// 2.2 Preserve current.config.json if it existed
		// -----------------------------------------------------------------
		if len(currentCfgBytes) > 0 {
			// Attempt to update the ID field similarly to main config
			if curAppCfg, err := domain.ParseAppConfig(currentCfgBytes); err == nil {
				curAppCfg.ID = newAppID
				if updated, err2 := json.MarshalIndent(curAppCfg, "", "  "); err2 == nil {
					currentCfgBytes = updated
				}
			}

			dstCurrentCfgPath := filepath.Join(newAppPath, "current.config.json")
			if err := os.WriteFile(dstCurrentCfgPath, currentCfgBytes, 0644); err != nil {
				log.Error("Failed to write preserved current.config.json", "path", dstCurrentCfgPath, "error", err)
			} else {
				log.Info("Preserved current configuration copy", "app_id", newAppID)
			}
		}

		// Prepare extension values: guarantee non-nil slice and deterministic order
		extVals := make([]domain.ExtensionValue, len(appCfg.ExtensionValues))
		copy(extVals, appCfg.ExtensionValues)

		// Sort by (extension, extension_app_id) to keep JSON output stable
		sort.Slice(extVals, func(i, j int) bool {
			if extVals[i].Extension == extVals[j].Extension {
				return extVals[i].ExtensionAppID < extVals[j].ExtensionAppID
			}
			return extVals[i].Extension < extVals[j].Extension
		})

		// Ensure the slice is non-nil even when empty so that JSON encodes as [] not null
		if extVals == nil {
			extVals = make([]domain.ExtensionValue, 0)
		}

		// Collect info for API call with cleaned extension values
		apps = append(apps, AppInfo{
			ID:              newAppID,
			TemplateID:      appCfg.TemplateID,
			Version:         appCfg.Version,
			Name:            appCfg.Name,
			Icon:            appCfg.Icon,
			Color:           appCfg.Color,
			ExtensionValues: extVals,
		})
	}

	// -----------------------------------------------------------------
	// 3.1 Second pass: update extension_values.extension_app_id references
	// -----------------------------------------------------------------
	for _, p := range processedApps {
		cfgPath := filepath.Join(p.newRevisionPath, "config.json")
		cfgBytes, err := os.ReadFile(cfgPath)
		if err != nil {
			log.Error("Failed to read config for extension update", "path", cfgPath, "error", err)
			continue
		}

		appCfg, err := domain.ParseAppConfig(cfgBytes)
		if err != nil {
			log.Error("Failed to parse app config for extension update", "path", cfgPath, "error", err)
			continue
		}

		updated := false
		for i := range appCfg.ExtensionValues {
			if newID, ok := oldToNewIDs[appCfg.ExtensionValues[i].ExtensionAppID]; ok {
				if newID != appCfg.ExtensionValues[i].ExtensionAppID {
					appCfg.ExtensionValues[i].ExtensionAppID = newID
					updated = true
				}
			}
		}

		if updated {
			newCfgBytes, err := json.MarshalIndent(appCfg, "", "  ")
			if err != nil {
				log.Error("Failed to marshal updated app config", "path", cfgPath, "error", err)
			} else if err := os.WriteFile(cfgPath, newCfgBytes, 0644); err != nil {
				log.Error("Failed to write updated app config", "path", cfgPath, "error", err)
			}
		}
	}

	// -----------------------------------------------------------------
	// 3.2 Update ExtensionValues in apps slice to use new IDs created
	//     in the first pass. Without this step, the restore payload
	//     may still reference obsolete application IDs because the
	//     apps slice was populated before cross-reference rewriting.
	// -----------------------------------------------------------------
	for i := range apps {
		for j := range apps[i].ExtensionValues {
			if newID, ok := oldToNewIDs[apps[i].ExtensionValues[j].ExtensionAppID]; ok {
				apps[i].ExtensionValues[j].ExtensionAppID = newID
			}
		}
	}

	// No apps found – nothing to send.
	if len(apps) == 0 {
		log.Info("No application templates found - restore finished")
		return nil
	}

	// ---------------------------------------------------------------------
	// 4. Create signed secret (agent_id + timestamp + apps)
	// ---------------------------------------------------------------------
	// Create deterministic representation of apps slice by sorting by app_id.
	sort.Slice(apps, func(i, j int) bool { return apps[i].ID < apps[j].ID })

	appsJSON, err := json.Marshal(apps)
	if err != nil {
		return fmt.Errorf("failed to marshal apps for signing: %w", err)
	}

	timestamp := time.Now().UTC().Format(time.RFC3339)
	message := []byte(cfg.AgentID + timestamp + string(appsJSON))

	secret, err := certs.SignWithPrivateKey(cfg.GetPrivateKeyPath(), message)
	if err != nil {
		return fmt.Errorf("failed to sign secret: %w", err)
	}

	// ---------------------------------------------------------------------
	// 5. Send request to backend
	// ---------------------------------------------------------------------
	payload := restoreDataRequest{
		AgentID:   cfg.AgentID,
		Timestamp: timestamp,
		Secret:    secret,
		Apps:      apps,
	}

	jsonBody, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal request payload: %w", err)
	}

	url := fmt.Sprintf("%s/api/v1/data/restore", cfg.GetAPIBaseURL())
	log.Info("Sending restore request", "url", url)

	httpClient := &http.Client{Timeout: 15 * time.Second}
	req, err := http.NewRequest("POST", url, bytes.NewReader(jsonBody))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("server responded with %d: %s", resp.StatusCode, string(body))
	}

	log.Info("Restore completed successfully")
	return nil
}

// copyDirectoryRecursive duplicates the entire src directory tree under dst.
// It preserves file modes but not ownership or timestamps (good enough for
// backup purposes). Existing dst will be overwritten if it already exists.
func copyDirectoryRecursive(src, dst string) error {
	// Ensure destination root exists.
	if err := os.MkdirAll(dst, 0755); err != nil {
		return err
	}

	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}

		targetPath := filepath.Join(dst, relPath)

		if info.IsDir() {
			return os.MkdirAll(targetPath, info.Mode())
		}

		// For regular files, copy contents.
		sourceFile, err := os.Open(path)
		if err != nil {
			return err
		}
		defer sourceFile.Close()

		// Ensure destination directory exists.
		if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
			return err
		}

		destFile, err := os.OpenFile(targetPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, info.Mode())
		if err != nil {
			return err
		}
		defer destFile.Close()

		if _, err := io.Copy(destFile, sourceFile); err != nil {
			return err
		}

		return nil
	})
}
