package save_app

import (
    "encoding/json"
    "fmt"
    "os"
    "path/filepath"
    "winterflow-agent/internal/domain/model"
    "winterflow-agent/pkg/certs"
    log "winterflow-agent/pkg/log"
    "winterflow-agent/pkg/yaml"
)

// SaveAppHandler handles the SaveAppCommand
type SaveAppHandler struct {
    AnsibleAppsRolesPath           string
    AnsibleAppsRolesCurrentVersion string
    PrivateKeyPath                 string
}

// Handle executes the SaveAppCommand
func (h *SaveAppHandler) Handle(cmd SaveAppCommand) error {
    if cmd.App == nil {
        return fmt.Errorf("app is nil in command")
    }
    app := cmd.App

    log.Printf("Processing save app request for app ID: %s", app.ID)

    // Resolve important directories once
    baseDir := filepath.Join(h.AnsibleAppsRolesPath, app.ID)
    versionDir := filepath.Join(baseDir, h.AnsibleAppsRolesCurrentVersion)
    dirs := map[string]string{
        "version":   versionDir,
        "defaults":  filepath.Join(versionDir, "defaults"),
        "vars":      filepath.Join(versionDir, "vars"),
        "templates": filepath.Join(versionDir, "templates"),
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

    // 3. Sync template files
    if err := h.syncTemplates(dirs["templates"], app.Config, app.Files); err != nil {
        return err
    }

    // 4. Write defaults from variables skeleton
    if err := h.writeDefaults(dirs["defaults"], app.Config); err != nil {
        return err
    }

    // 5. Write vars YAML file (secrets are stored together with regular variables)
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
    // Build sets for quick look-ups
    expected := make(map[string]model.AppFile) // filename -> AppFile
    idToFilename := make(map[string]string)
    for _, f := range cfg.Files {
        expected[f.Filename] = f
        idToFilename[f.ID] = f.Filename
    }

    // Remove obsolete .j2 files
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

    // Write/update templates provided in the request
    for id, content := range contentMap {
        filename, ok := idToFilename[id]
        if !ok {
            log.Warn("No filename found for file ID %s, skipping", id)
            continue
        }

        targetPath := filepath.Join(templatesDir, filename+".j2")
        if err := os.WriteFile(targetPath, content, 0o644); err != nil {
            return fmt.Errorf("error writing template %s: %w", targetPath, err)
        }
        log.Debug("Wrote template: %s", targetPath)
    }

    return nil
}

// writeDefaults produces defaults/main.yml with empty values for every variable in the config.
func (h *SaveAppHandler) writeDefaults(defaultsDir string, cfg *model.AppConfig) error {
    empty := make(map[string]string)
    for _, v := range cfg.Variables {
        empty[v.Name] = ""
    }

    j, err := json.Marshal(empty)
    if err != nil {
        return fmt.Errorf("error marshaling defaults JSON: %w", err)
    }
    y, err := yaml.JSONToYAML(j)
    if err != nil {
        return fmt.Errorf("error converting defaults to YAML: %w", err)
    }

    path := filepath.Join(defaultsDir, "main.yml")
    if err := os.WriteFile(path, y, 0o644); err != nil {
        return fmt.Errorf("error writing defaults file: %w", err)
    }
    return nil
}

// writeVars writes all variables (including encrypted ones) into vars.yml.
func (h *SaveAppHandler) writeVars(varsDir string, cfg *model.AppConfig, input model.VariableMap) error {
    varsFile := filepath.Join(varsDir, "vars.yml")

    // Load existing values to preserve secrets when placeholder "<encrypted>" is passed.
    existingVars := make(map[string]string)
    if data, err := os.ReadFile(varsFile); err == nil {
        _ = yaml.UnmarshalYAML(data, &existingVars)
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

            // Attempt to decrypt before storing so the role gets plain text.
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

    // Convert to YAML and write the file.
    j, err := json.Marshal(vars)
    if err != nil {
        return fmt.Errorf("error marshaling vars JSON: %w", err)
    }
    y, err := yaml.JSONToYAML(j)
    if err != nil {
        return fmt.Errorf("error converting vars to YAML: %w", err)
    }
    if err := os.WriteFile(varsFile, y, 0o644); err != nil {
        return fmt.Errorf("error writing vars file: %w", err)
    }

    return nil
}

// SaveAppResult represents the result of creating an app
type SaveAppResult struct {
    Success         bool
    ResponseMessage string
    App             *model.App
}

// NewSaveAppHandler creates a new SaveAppHandler
func NewSaveAppHandler(ansibleAppsRolesPath, ansibleAppsRolesCurrentVersion, privateKeyPath string) *SaveAppHandler {
    return &SaveAppHandler{
        AnsibleAppsRolesPath:           ansibleAppsRolesPath,
        AnsibleAppsRolesCurrentVersion: ansibleAppsRolesCurrentVersion,
        PrivateKeyPath:                 privateKeyPath,
    }
}
