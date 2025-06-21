package get_app

import (
	"fmt"
	"os"
	"path/filepath"
	"winterflow-agent/internal/domain/model"
	log "winterflow-agent/pkg/log"
)

// GetAppQueryHandler handles the GetAppQuery
type GetAppQueryHandler struct {
	AnsibleAppsRolesPath           string
	AnsibleAppsRolesCurrentVersion string
}

// Handle executes the GetAppQuery and returns the result
func (h *GetAppQueryHandler) Handle(query GetAppQuery) (*model.App, error) {
	log.Printf("Processing get app request for app ID: %s", query.AppID)

	appID := query.AppID

	// 1. Determine version directory
	versionDir := h.AnsibleAppsRolesCurrentVersion
	if query.AppVersion > 0 {
		versionDir = fmt.Sprintf("%d", query.AppVersion)
	}

	rolesDir := filepath.Join(h.AnsibleAppsRolesPath, appID, versionDir)
	rolesVarsDir := filepath.Join(rolesDir, "vars")
	rolesTemplatesDir := filepath.Join(rolesDir, "templates")

	if _, err := os.Stat(rolesDir); os.IsNotExist(err) {
		return nil, fmt.Errorf("app with ID %s not found", appID)
	}

	// 2. Load and parse config
	configPath := filepath.Join(rolesDir, "config.json")
	configBytes, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("error reading config file: %w", err)
	}
	appConfig, err := model.ParseAppConfig(configBytes)
	if err != nil {
		return nil, fmt.Errorf("error parsing config: %w", err)
	}

	// 3. Build variables map
	varsMap, err := h.loadVariables(appConfig, configBytes, rolesVarsDir)
	if err != nil {
		return nil, err
	}

	// 4. Build files map
	filesMap, err := h.loadFiles(appConfig, rolesTemplatesDir)
	if err != nil {
		return nil, err
	}

	// 5. Return App model
	return &model.App{
		ID:        appID,
		Config:    appConfig,
		Variables: varsMap,
		Files:     filesMap,
	}, nil
}

// loadVariables builds the final VariableMap taking into account encryption flags.
func (h *GetAppQueryHandler) loadVariables(appConfig *model.AppConfig, configBytes []byte, varsDir string) (model.VariableMap, error) {
	varsFilePath := filepath.Join(varsDir, "vars.yml")
	varsBytes, err := os.ReadFile(varsFilePath)
	if err != nil {
		return nil, fmt.Errorf("error reading vars file: %w", err)
	}

	secretsFilePath := filepath.Join(varsDir, "secrets.yml")
	secretsBytes, err := os.ReadFile(secretsFilePath)
	if err != nil {
		return nil, fmt.Errorf("error reading secrets file: %w", err)
	}

	// non-encrypted values from vars.yml
	varsJSON, err := convertYAMLToIDValueJSON(configBytes, varsBytes)
	if err != nil {
		return nil, fmt.Errorf("error converting vars YAML: %w", err)
	}
	variables, err := convertJSONToVariableMap(varsJSON)
	if err != nil {
		return nil, fmt.Errorf("error converting vars JSON to map: %w", err)
	}

	// encrypted values from secrets.yml
	secretsJSON, err := convertYAMLToIDValueJSON(configBytes, secretsBytes)
	if err != nil {
		return nil, fmt.Errorf("error converting secrets YAML: %w", err)
	}
	secrets, err := convertJSONToEncryptedVariableMap(secretsJSON)
	if err != nil {
		return nil, fmt.Errorf("error converting secrets JSON to map: %w", err)
	}

	// merge secrets over vars (encrypted wins)
	for id, v := range secrets {
		variables[id] = v
	}

	// finally, honour IsEncrypted flag from the config itself (in case value is stored in vars.yml)
	for _, v := range appConfig.Variables {
		if v.IsEncrypted {
			variables[v.ID] = "<encrypted>"
		}
	}

	return variables, nil
}

// loadFiles reads template files according to the config and applies encryption flag.
func (h *GetAppQueryHandler) loadFiles(appConfig *model.AppConfig, templatesDir string) (map[string][]byte, error) {
	files := make(map[string][]byte)

	for _, f := range appConfig.Files {
		if f.IsEncrypted {
			files[f.ID] = []byte("<encrypted>")
			continue
		}

		filePath := filepath.Join(templatesDir, f.Filename+".j2")
		content, err := os.ReadFile(filePath)
		if err != nil {
			// If the file is missing we log and continue – it might be optional.
			log.Printf("Warning: template file %s not found: %v", filePath, err)
			continue
		}
		files[f.ID] = content
	}

	return files, nil
}

// NewGetAppQueryHandler creates a new GetAppQueryHandler
func NewGetAppQueryHandler(ansibleAppsRolesPath, ansibleAppsRolesCurrentVersion string) *GetAppQueryHandler {
	return &GetAppQueryHandler{
		AnsibleAppsRolesPath:           ansibleAppsRolesPath,
		AnsibleAppsRolesCurrentVersion: ansibleAppsRolesCurrentVersion,
	}
}
