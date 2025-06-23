package save_app

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"winterflow-agent/internal/domain/model"
	"winterflow-agent/internal/domain/service/app"
	"winterflow-agent/pkg/certs"
	log "winterflow-agent/pkg/log"
)

// SaveAppHandler handles the SaveAppCommand
type SaveAppHandler struct {
	AppsTemplatesPath string
	PrivateKeyPath    string
	versionService    app.AppVersionServiceInterface
}

// Handle executes the SaveAppCommand
func (h *SaveAppHandler) Handle(cmd SaveAppCommand) error {
	if cmd.App == nil {
		return fmt.Errorf("app is nil in command")
	}
	app := cmd.App

	log.Printf("Processing save app request for app ID: %s", app.ID)

	// Prevent renaming: if the application already exists, always keep the original name stored on disk.
	baseDir := filepath.Join(h.AppsTemplatesPath, app.ID)

	// Ensure the base directory for the application exists. This is required so that subsequent
	// operations (like reading a previous config or creating version directories) do not fail
	// due to a missing parent path.
	if err := os.MkdirAll(baseDir, 0o755); err != nil {
		return fmt.Errorf("error creating base directory %s: %w", baseDir, err)
	}

	// -------------------------------------------------------------------
	// Version handling – always create a fresh version before applying the
	// incoming changes. If a version service is configured, we rely on it to
	// duplicate the latest version (thereby preserving the existing state).
	// If it is not available we fall back to the configured current version
	// folder (legacy behaviour).
	// -------------------------------------------------------------------
	if h.versionService == nil {
		return fmt.Errorf("version service is not configured for SaveAppHandler")
	}

	newVersion, err := h.versionService.CreateVersion(app.ID)
	if err != nil {
		return fmt.Errorf("failed to create new version for app %s: %w", app.ID, err)
	}

	// Use the service helpers to construct version specific paths
	versionDir := h.versionService.GetVersionDir(app.ID, newVersion)
	log.Debug("Created new version %d for app %s", newVersion, app.ID)

	existingCfgPath := filepath.Join(versionDir, "config.json")
	var prevFiles []model.AppFile
	if data, err := os.ReadFile(existingCfgPath); err == nil {
		if existingCfg, err := model.ParseAppConfig(data); err == nil {
			// If the stored name differs from the incoming one, keep the old name.
			storedName := strings.TrimSpace(existingCfg.Name)
			incomingName := strings.TrimSpace(app.Config.Name)
			if storedName != "" && !strings.EqualFold(storedName, incomingName) {
				log.Info("[SaveApp] Rename attempted from '%s' to '%s' for app ID %s – keeping the original name", incomingName, storedName, app.ID)
				app.Config.Name = existingCfg.Name
			}

			// Preserve previous files metadata for rename detection
			prevFiles = existingCfg.Files
		}
	}

	// Validate that the (possibly overridden) application name is provided and unique
	if strings.TrimSpace(app.Config.Name) == "" {
		return fmt.Errorf("application name cannot be empty")
	}

	unique, err := h.isNameUnique(app.Config.Name, app.ID)
	if err != nil {
		return err
	}
	if !unique {
		return fmt.Errorf("application name '%s' is already in use by another app", app.Config.Name)
	}

	// Resolve important directories once (baseDir & versionDir already calculated above)
	dirs := map[string]string{
		"version": versionDir,
		"vars":    h.versionService.GetVarsDir(app.ID, newVersion),
		"files":   h.versionService.GetFilesDir(app.ID, newVersion),
	}

	// 1. Ensure directory structure exists
	for _, d := range dirs {
		if err := os.MkdirAll(d, 0o755); err != nil {
			return fmt.Errorf("error creating directory %s: %w", d, err)
		}
	}

	// 2. Persist config.json
	if err := h.writeConfig(dirs["version"], app.Config); err != nil {
		return err
	}

	// 3. Sync template files to /files directory
	if err := h.syncTemplates(dirs["files"], app.Config, prevFiles, app.Files); err != nil {
		return err
	}

	// 4. Write vars JSON file (secrets are stored together with regular variables)
	if err := h.writeVars(dirs["vars"], app.Config, app.Variables); err != nil {
		return err
	}

	// 5. Clean up old versions if we have a version service
	if h.versionService != nil {
		if err := h.versionService.DeleteOldVersions(app.ID); err != nil {
			log.Warn("Failed to clean up old versions for app %s: %v", app.ID, err)
			// Don't fail the save operation if cleanup fails
		} else {
			log.Debug("Successfully cleaned up old versions for app %s", app.ID)
		}
	}

	return nil
}

// writeConfig marshals the AppConfig and writes it to config.json inside versionDir.
func (h *SaveAppHandler) writeConfig(versionDir string, cfg *model.AppConfig) error {
	configPath := filepath.Join(versionDir, "config.json")
	data, err := json.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("error marshaling app config: %w", err)
	}
	if err := os.WriteFile(configPath, data, 0o644); err != nil {
		return fmt.Errorf("error writing config file: %w", err)
	}
	return nil
}

// syncTemplates keeps the templates directory in sync with cfg.Files and contentMap.
func (h *SaveAppHandler) syncTemplates(templatesDir string, cfg *model.AppConfig, prevFiles []model.AppFile, contentMap model.FilesMap) error {
	// Build helper maps for quick look-ups.
	expected := make(map[string]model.AppFile) // filename (as provided in cfg) -> AppFile
	idToFile := make(map[string]model.AppFile) // file ID -> AppFile
	prevIDToFile := make(map[string]model.AppFile)

	for _, f := range cfg.Files {
		expected[f.Filename] = f
		idToFile[f.ID] = f
	}
	for _, f := range prevFiles {
		prevIDToFile[f.ID] = f
	}

	// ---------------------------------------------------------------------
	// 0. Handle renames when file content is not provided (encrypted placeholder).
	// ---------------------------------------------------------------------
	for id, newMeta := range idToFile {
		prevMeta, ok := prevIDToFile[id]
		if !ok {
			continue // new file or unknown – nothing to rename
		}

		if prevMeta.Filename == newMeta.Filename {
			continue // same path – nothing to rename
		}

		// If the client sends placeholder for encrypted file, we need to move/copy
		content, ok := contentMap[id]
		if !ok || string(content) != "<encrypted>" {
			// Either content provided or not encrypted – regular flow will handle
			continue
		}

		// Compute old and new paths.
		oldRel := filepath.FromSlash(prevMeta.Filename)
		oldRel = strings.TrimLeft(oldRel, string(os.PathSeparator))
		oldPath := filepath.Join(templatesDir, oldRel+".j2")

		newRel := filepath.FromSlash(newMeta.Filename)
		newRel = strings.TrimLeft(newRel, string(os.PathSeparator))
		newPath := filepath.Join(templatesDir, newRel+".j2")

		// Ensure target directory.
		if err := os.MkdirAll(filepath.Dir(newPath), 0o755); err != nil {
			return fmt.Errorf("error creating directories for %s: %w", newPath, err)
		}

		// Copy file bytes.
		data, err := os.ReadFile(oldPath)
		if err != nil {
			log.Warn("Failed to read source file %s for rename: %v", oldPath, err)
			continue
		}

		if err := os.WriteFile(newPath, data, 0o644); err != nil {
			return fmt.Errorf("error writing renamed template %s: %w", newPath, err)
		}
		log.Debug("Copied template from %s to %s (rename)", oldPath, newPath)
	}

	// ---------------------------------------------------------------------
	// 1. Delete obsolete templates (recursive walk).
	// ---------------------------------------------------------------------
	// Build a set with expected absolute paths (with .j2 suffix).
	expectedPaths := make(map[string]struct{})
	for filename := range expected {
		// Ensure filename is always treated as a relative path (no leading separators)
		rel := filepath.FromSlash(filename)
		rel = strings.TrimLeft(rel, string(os.PathSeparator))
		relPath := rel + ".j2"
		expectedPaths[filepath.Clean(filepath.Join(templatesDir, relPath))] = struct{}{}
	}

	// Walk through existing files and remove any .j2 that is not expected.
	if err := filepath.WalkDir(templatesDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if !strings.HasSuffix(d.Name(), ".j2") {
			return nil
		}

		if _, ok := expectedPaths[filepath.Clean(path)]; !ok {
			if err := os.Remove(path); err != nil {
				return fmt.Errorf("error removing obsolete template %s: %w", path, err)
			}
			log.Debug("Deleted obsolete template: %s", path)

			// Attempt to clean up empty directories up the chain.
			dir := filepath.Dir(path)
			for dir != templatesDir {
				entries, _ := os.ReadDir(dir)
				if len(entries) == 0 {
					_ = os.Remove(dir)
					dir = filepath.Dir(dir)
				} else {
					break
				}
			}
		}
		return nil
	}); err != nil {
		return err
	}

	// ---------------------------------------------------------------------
	// 2. Write / update templates sent in the request.
	// ---------------------------------------------------------------------
	for id, content := range contentMap {
		fileMeta, ok := idToFile[id]
		if !ok {
			log.Warn("No metadata found for file ID %s, skipping", id)
			continue
		}

		relFilename := filepath.FromSlash(fileMeta.Filename)
		relFilename = strings.TrimLeft(relFilename, string(os.PathSeparator))
		targetPath := filepath.Join(templatesDir, relFilename+".j2")

		// Ensure the directory for the file exists.
		if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
			return fmt.Errorf("error creating directories for %s: %w", targetPath, err)
		}

		// Handle encrypted files.
		if fileMeta.IsEncrypted {
			// Handle placeholder without breaking creation of brand-new encrypted files.
			if string(content) == "<encrypted>" {
				if _, err := os.Stat(targetPath); err == nil {
					// File already exists – nothing to overwrite.
					log.Debug("Placeholder received for encrypted file %s (ID: %s), keeping existing file", fileMeta.Filename, id)
					continue
				}

				// New file with placeholder – create an empty stub so that path exists on disk.
				if err := os.WriteFile(targetPath, []byte("<encrypted>"), 0o644); err != nil {
					return fmt.Errorf("error writing placeholder template %s: %w", targetPath, err)
				}
				log.Debug("Created placeholder for new encrypted file: %s", targetPath)
				continue
			}

			plaintext := content
			if h.PrivateKeyPath != "" {
				if dec, err := certs.DecryptWithPrivateKey(h.PrivateKeyPath, string(content)); err == nil {
					plaintext = []byte(dec)
				} else {
					log.Warn("Failed to decrypt file %s: %v", fileMeta.Filename, err)
				}
			}

			if err := os.WriteFile(targetPath, plaintext, 0o644); err != nil {
				return fmt.Errorf("error writing template %s: %w", targetPath, err)
			}
			log.Debug("Wrote (decrypted) template: %s", targetPath)
			continue
		}

		// Non-encrypted files – write content as-is.
		if err := os.WriteFile(targetPath, content, 0o644); err != nil {
			return fmt.Errorf("error writing template %s: %w", targetPath, err)
		}
		log.Debug("Wrote template: %s", targetPath)
	}

	return nil
}

// writeVars writes all variables (including encrypted ones) into vars/values.json.
func (h *SaveAppHandler) writeVars(varsDir string, cfg *model.AppConfig, input model.VariableMap) error {
	varsFile := filepath.Join(varsDir, "values.json")

	// Load existing values to preserve secrets when placeholder "<encrypted>" is passed.
	existingVars := make(map[string]string)
	if data, err := os.ReadFile(varsFile); err == nil {
		_ = json.Unmarshal(data, &existingVars)
	}

	// Prepare resulting map keyed by variable name.
	vars := make(map[string]string)

	for _, v := range cfg.Variables {
		value, ok := input[v.ID]
		if !ok {
			// No value in request – keep existing one if present.
			if existing, ok := existingVars[v.Name]; ok {
				vars[v.Name] = existing
			}
			continue
		}

		// Handle encrypted variables.
		if v.IsEncrypted {
			if value == "<encrypted>" {
				// Preserve existing value (if any) or use empty string to keep key present.
				if existing, ok := existingVars[v.Name]; ok {
					vars[v.Name] = existing
				} else {
					vars[v.Name] = ""
				}
			} else {
				vars[v.Name] = value
			}

			// Attempt to decrypt before storing so the consumer gets plain text.
			if h.PrivateKeyPath != "" && vars[v.Name] != "" {
				dec, err := certs.DecryptWithPrivateKey(h.PrivateKeyPath, vars[v.Name])
				if err == nil {
					vars[v.Name] = dec
				} else {
					log.Warn("Failed to decrypt variable %s: %v", v.Name, err)
				}
			}
		} else {
			// Plain variable, just store the provided value.
			vars[v.Name] = value
		}
	}

	// Convert to JSON and write the file.
	j, err := json.MarshalIndent(vars, "", "  ")
	if err != nil {
		return fmt.Errorf("error marshaling vars JSON: %w", err)
	}
	if err := os.WriteFile(varsFile, j, 0o644); err != nil {
		return fmt.Errorf("error writing vars file: %w", err)
	}

	return nil
}

// isNameUnique checks that the given application name is not used by any other application (different appID).
func (h *SaveAppHandler) isNameUnique(name string, currentAppID string) (bool, error) {
	entries, err := os.ReadDir(h.AppsTemplatesPath)
	if err != nil {
		return false, fmt.Errorf("failed to read apps templates directory: %w", err)
	}

	for _, e := range entries {
		if !e.IsDir() {
			continue
		}

		appID := e.Name()
		if appID == currentAppID {
			// Skip the current app (we allow renaming within same ID)
			continue
		}

		// Resolve the latest version for the application so we always check the most up-to-date config.
		latestVersion, err := h.versionService.GetLatestAppVersion(appID)
		if err != nil {
			// If we cannot determine the latest version, skip this application – not critical.
			continue
		}
		if latestVersion == 0 {
			// Application does not have any versions yet (should not normally happen).
			continue
		}

		cfgPath := filepath.Join(h.versionService.GetVersionDir(appID, latestVersion), "config.json")
		data, err := os.ReadFile(cfgPath)
		if err != nil {
			continue // ignore missing configs or read errors – not critical for uniqueness check
		}

		cfg, err := model.ParseAppConfig(data)
		if err != nil {
			continue // skip invalid configs
		}

		if strings.EqualFold(strings.TrimSpace(cfg.Name), strings.TrimSpace(name)) {
			return false, nil
		}
	}

	return true, nil
}

// SaveAppResult represents the result of creating an app
type SaveAppResult struct {
	Success         bool
	ResponseMessage string
	App             *model.App
}

// NewSaveAppHandler creates a new SaveAppHandler
func NewSaveAppHandler(appsTemplatesPath, privateKeyPath string, versionService app.AppVersionServiceInterface) *SaveAppHandler {
	return &SaveAppHandler{
		AppsTemplatesPath: appsTemplatesPath,
		PrivateKeyPath:    privateKeyPath,
		versionService:    versionService,
	}
}
