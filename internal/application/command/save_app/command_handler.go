package save_app

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"winterflow-agent/internal/domain/model"
	"winterflow-agent/pkg/certs"
	log "winterflow-agent/pkg/log"
)

// SaveAppHandler handles the SaveAppCommand
type SaveAppHandler struct {
	AppsTemplatesPath  string
	AppsCurrentVersion string
	PrivateKeyPath     string
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
	versionDir := filepath.Join(baseDir, h.AppsCurrentVersion)

	existingCfgPath := filepath.Join(versionDir, "config.json")
	if data, err := os.ReadFile(existingCfgPath); err == nil {
		if existingCfg, err := model.ParseAppConfig(data); err == nil {
			// If the stored name differs from the incoming one, keep the old name.
			storedName := strings.TrimSpace(existingCfg.Name)
			incomingName := strings.TrimSpace(app.Config.Name)
			if storedName != "" && !strings.EqualFold(storedName, incomingName) {
				log.Info("[SaveApp] Rename attempted from '%s' to '%s' for app ID %s – keeping the original name", incomingName, storedName, app.ID)
				app.Config.Name = existingCfg.Name
			}
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
		"vars":    filepath.Join(versionDir, "vars"),
		"files":   filepath.Join(versionDir, "files"),
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
	if err := h.syncTemplates(dirs["files"], app.Config, app.Files); err != nil {
		return err
	}

	// 4. Write vars JSON file (secrets are stored together with regular variables)
	if err := h.writeVars(dirs["vars"], app.Config, app.Variables); err != nil {
		return err
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
func (h *SaveAppHandler) syncTemplates(templatesDir string, cfg *model.AppConfig, contentMap model.FilesMap) error {
	// Build sets for quick look-ups and helper maps for encryption handling.
	expected := make(map[string]model.AppFile) // filename -> AppFile
	idToFile := make(map[string]model.AppFile) // file ID -> full AppFile (for IsEncrypted flag)

	for _, f := range cfg.Files {
		expected[f.Filename] = f
		idToFile[f.ID] = f
	}

	// Remove obsolete .j2 files inside /files directory
	existing, err := filepath.Glob(filepath.Join(templatesDir, "*.j2"))
	if err != nil {
		return fmt.Errorf("error listing template files: %w", err)
	}
	for _, path := range existing {
		name := filepath.Base(path)
		name = name[:len(name)-3] // strip .j2
		if _, ok := expected[name]; !ok {
			if err := os.Remove(path); err != nil {
				return fmt.Errorf("error removing obsolete template %s: %w", path, err)
			}
			log.Debug("Deleted obsolete template: %s", path)
		}
	}

	// Write/update templates provided in the request respecting encryption flags.
	for id, content := range contentMap {
		fileMeta, ok := idToFile[id]
		if !ok {
			log.Warn("No metadata found for file ID %s, skipping", id)
			continue
		}

		filename := fileMeta.Filename
		targetPath := filepath.Join(templatesDir, filename+".j2")

		// Handle encrypted files.
		if fileMeta.IsEncrypted {
			plaintext := []byte{}

			// If the placeholder is passed, we keep the existing file unchanged.
			if string(content) == "<encrypted>" {
				log.Debug("Placeholder received for encrypted file %s (ID: %s), keeping existing file", filename, id)
				continue
			}

			// Attempt decryption using agent's private key, if available.
			if h.PrivateKeyPath != "" {
				dec, err := certs.DecryptWithPrivateKey(h.PrivateKeyPath, string(content))
				if err != nil {
					log.Warn("Failed to decrypt file %s: %v", filename, err)
					// Fallback: store the original encrypted payload to disk to avoid data loss.
					plaintext = content
				} else {
					plaintext = []byte(dec)
				}
			} else {
				// No private key – store encrypted payload as-is.
				plaintext = content
			}

			if err := os.WriteFile(targetPath, plaintext, 0o644); err != nil {
				return fmt.Errorf("error writing template %s: %w", targetPath, err)
			}
			log.Debug("Wrote (decrypted) template: %s", targetPath)
			continue
		}

		// Non-encrypted file – write content as-is.
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

		cfgPath := filepath.Join(h.AppsTemplatesPath, appID, h.AppsCurrentVersion, "config.json")
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
func NewSaveAppHandler(appsTemplatesPath, appsCurrentVersion, privateKeyPath string) *SaveAppHandler {
	return &SaveAppHandler{
		AppsTemplatesPath:  appsTemplatesPath,
		AppsCurrentVersion: appsCurrentVersion,
		PrivateKeyPath:     privateKeyPath,
	}
}
