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

// AppInfo mirrors the server-side structure used by /api/v1/data/restore.
// It is duplicated locally to avoid importing server packages.
type AppInfo struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Icon  string `json:"icon"`
	Color string `json:"color"`
}

// restoreDataRequest matches the payload expected by the backend.
type restoreDataRequest struct {
	AgentID   string    `json:"agent_id"`
	Timestamp string    `json:"timestamp"`
	Secret    string    `json:"secret"`
	Apps      []AppInfo `json:"apps"`
}

// RestoreAgentData scans the local apps_templates folder, regenerates UUIDs,
// keeps only the latest version for every app, updates the config.json files
// accordingly and notifies the WinterFlow backend via /api/v1/data/restore.
//
// It is intended to be executed via `winterflow-agent --restore` after the
// agent has been re-installed on a server while preserving the data volume.
func RestoreAgentData(configPath string) error {
	fmt.Println("Starting restore procedure…")

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

	fmt.Println("Creating backup of application templates", "source", templatesRoot, "destination", backupRoot)
	if err := copyDirectoryRecursive(templatesRoot, backupRoot); err != nil {
		return fmt.Errorf("failed to create backup: %w", err)
	}
	fmt.Println("Backup created successfully", "path", backupRoot)

	// ---------------------------------------------------------------------
	// 3. Iterate over apps_templates and rewrite structure
	// ---------------------------------------------------------------------
	entries, err := os.ReadDir(templatesRoot)
	if err != nil {
		return fmt.Errorf("cannot read apps_templates directory %s: %w", templatesRoot, err)
	}

	var apps []AppInfo

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		oldAppID := entry.Name()
		oldAppPath := filepath.Join(templatesRoot, oldAppID)

		// Determine latest version subdirectory (highest numeric name).
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

		// Generate new UUID for the app.
		newAppID := uuid.New().String()

		newAppPath := filepath.Join(templatesRoot, newAppID)
		newVersionPath := filepath.Join(newAppPath, "1")

		// Make sure parent directory exists.
		if err := os.MkdirAll(newAppPath, 0755); err != nil {
			log.Error("Failed to create new app directory", "path", newAppPath, "error", err)
			continue
		}

		// Move (rename) latest version directory to the new location.
		src := filepath.Join(oldAppPath, latestDirName)
		if err := os.Rename(src, newVersionPath); err != nil {
			log.Error("Failed to move version directory", "src", src, "dst", newVersionPath, "error", err)
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
		cfgPath := filepath.Join(newVersionPath, "config.json")
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
				fmt.Println("Preserved current configuration copy", "app_id", newAppID)
			}
		}

		// Collect info for API call.
		apps = append(apps, AppInfo{
			ID:    newAppID,
			Name:  appCfg.Name,
			Icon:  appCfg.Icon,
			Color: appCfg.Color,
		})
	}

	// No apps found – nothing to send.
	if len(apps) == 0 {
		fmt.Errorf("No application templates found – restore finished")
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
	fmt.Println("Sending restore request", "url", url, "payload", string(jsonBody))

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

	fmt.Println("Restore completed successfully")
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
