package get_app

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"winterflow-agent/internal/domain/model"
	"winterflow-agent/internal/domain/service/app"
	"winterflow-agent/pkg/log"
)

// GetAppQueryHandler handles the GetAppQuery
type GetAppQueryHandler struct {
	VersionService app.RevisionServiceInterface
}

// Handle executes the GetAppQuery and returns the result
func (h *GetAppQueryHandler) Handle(query GetAppQuery) (*model.AppDetails, error) {
	log.Info("Processing get app request", "app_id", query.AppID)

	appID := query.AppID

	// 1. Determine version directory
	var targetVersion uint32
	if query.AppRevision > 0 {
		// A specific version was requested by the caller. Validate that it exists.
		exists, err := h.VersionService.ValidateAppRevision(appID, query.AppRevision)
		if err != nil {
			return nil, fmt.Errorf("error validating app version: %w", err)
		}
		if !exists {
			return nil, fmt.Errorf("version %d not found for app %s", query.AppRevision, appID)
		}

		targetVersion = query.AppRevision
	} else {
		// No version supplied – resolve to the latest available version using the service.
		latest, err := h.VersionService.GetLatestAppRevision(appID)
		if err != nil {
			return nil, fmt.Errorf("error determining latest version for app %s: %w", appID, err)
		}
		if latest == 0 {
			return nil, fmt.Errorf("no versions found for app %s", appID)
		}
		targetVersion = latest
	}

	templatesDir := h.VersionService.GetRevisionDir(appID, targetVersion)
	varsDir := h.VersionService.GetVarsDir(appID, targetVersion)
	filesDir := h.VersionService.GetFilesDir(appID, targetVersion)

	if _, err := os.Stat(templatesDir); os.IsNotExist(err) {
		return nil, fmt.Errorf("app with ID %s not found", appID)
	}

	// 2. Load and parse config
	configPath := filepath.Join(templatesDir, "config.json")
	configBytes, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("error reading config file: %w", err)
	}
	appConfig, err := model.ParseAppConfig(configBytes)
	if err != nil {
		return nil, fmt.Errorf("error parsing config: %w", err)
	}

	// 3. Build variables map
	varsMap, err := h.loadVariables(appConfig, varsDir)
	if err != nil {
		return nil, err
	}

	// 4. Build files map
	filesMap, err := h.loadFiles(appConfig, filesDir)
	if err != nil {
		return nil, err
	}

	// 5. Get all versions for the app
	versions, err := h.VersionService.GetAppRevisions(appID)
	if err != nil {
		return nil, err
	}

	// 6. Return App model
	return &model.AppDetails{
		App: &model.App{
			ID:        appID,
			Config:    appConfig,
			Variables: varsMap,
			Files:     filesMap,
		},
		Revision:  targetVersion,
		Revisions: versions,
	}, nil
}

// loadVariables builds the final VariableMap taking into account encryption flags.
func (h *GetAppQueryHandler) loadVariables(appConfig *model.AppConfig, varsDir string) (model.VariableMap, error) {
	varsFilePath := filepath.Join(varsDir, "values.json")
	varsBytes, err := os.ReadFile(varsFilePath)
	if err != nil {
		return nil, fmt.Errorf("error reading vars file: %w", err)
	}

	// Parse JSON into generic map
	var raw map[string]interface{}
	if err := json.Unmarshal(varsBytes, &raw); err != nil {
		return nil, fmt.Errorf("error parsing vars JSON: %w", err)
	}

	// Build name -> ID map from config
	nameToID := make(map[string]string)
	for _, v := range appConfig.Variables {
		nameToID[v.Name] = v.ID
	}

	variables := make(model.VariableMap)
	for k, v := range raw {
		id, ok := nameToID[k]
		if !ok {
			// Key might already be ID, use as is
			id = k
		}

		// Convert value to string
		variables[id] = fmt.Sprintf("%v", v)
	}

	// For encrypted variables ensure the placeholder is returned instead of the real value.
	for _, v := range appConfig.Variables {
		if v.IsEncrypted {
			variables[v.ID] = "<encrypted>"
		}
	}

	return variables, nil
}

// loadFiles reads template files according to the config and applies encryption flag.
func (h *GetAppQueryHandler) loadFiles(appConfig *model.AppConfig, filesDir string) (map[string][]byte, error) {
	files := make(map[string][]byte)

	for _, f := range appConfig.Files {
		if f.IsEncrypted {
			files[f.ID] = []byte("<encrypted>")
			continue
		}

		filePath := filepath.Join(filesDir, f.Name)
		content, err := os.ReadFile(filePath)
		if err != nil {
			// If the file is missing we log and continue – it might be optional.
			log.Warn("Template file not found", "file_path", filePath, "error", err)
			continue
		}
		files[f.ID] = content
	}

	return files, nil
}

// NewGetAppQueryHandler creates a new GetAppQueryHandler
func NewGetAppQueryHandler(versionService app.RevisionServiceInterface) *GetAppQueryHandler {
	return &GetAppQueryHandler{
		VersionService: versionService,
	}
}
