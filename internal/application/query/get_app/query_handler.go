package get_app

import (
	"encoding/json"
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

	// Get the app ID from the query
	appID := query.AppID

	// Determine the app version to use
	var versionDir string
	if query.AppVersion > 0 {
		versionDir = fmt.Sprintf("%d", query.AppVersion)
	} else {
		versionDir = h.AnsibleAppsRolesCurrentVersion
	}

	// Define the paths to the app files
	rolesDir := filepath.Join(h.AnsibleAppsRolesPath, appID, versionDir)
	rolesVarsDir := filepath.Join(rolesDir, "vars")
	rolesTemplatesDir := filepath.Join(rolesDir, "templates")

	// Check if the app directory exists
	if _, err := os.Stat(rolesDir); os.IsNotExist(err) {
		return nil, fmt.Errorf("app with ID %s not found", appID)
	}

	// Read the config file
	configFile := filepath.Join(rolesDir, "config.json")
	configBytes, err := os.ReadFile(configFile)
	if err != nil {
		return nil, fmt.Errorf("error reading config file: %w", err)
	}

	// Read the variables file
	varsFile := filepath.Join(rolesVarsDir, "vars.yml")
	varsBytes, err := os.ReadFile(varsFile)
	if err != nil {
		return nil, fmt.Errorf("error reading vars file: %w", err)
	}

	// Read the secrets file
	secretsFile := filepath.Join(rolesVarsDir, "secrets.yml")
	secretsBytes, err := os.ReadFile(secretsFile)
	if err != nil {
		return nil, fmt.Errorf("error reading secrets file: %w", err)
	}

	// Convert variables from YAML to JSON with "id": "value" format
	varsJSON, err := convertYAMLToIDValueJSON(configBytes, varsBytes)
	if err != nil {
		return nil, fmt.Errorf("error converting variables to JSON: %w", err)
	}

	// Convert variables JSON to variable map
	variables, err := convertJSONToVariableMap(varsJSON)
	if err != nil {
		return nil, fmt.Errorf("error converting variables JSON to variable map: %w", err)
	}

	// Convert secrets from YAML to JSON with "id": "value" format
	secretsJSON, err := convertYAMLToIDValueJSON(configBytes, secretsBytes)
	if err != nil {
		return nil, fmt.Errorf("error converting secrets to JSON: %w", err)
	}

	// Convert secrets JSON to variable map with encrypted values
	secrets, err := convertJSONToEncryptedVariableMap(secretsJSON)
	if err != nil {
		return nil, fmt.Errorf("error converting secrets JSON to encrypted variable map: %w", err)
	}

	// Parse config to get list of files
	var configData map[string]interface{}
	if err := json.Unmarshal(configBytes, &configData); err != nil {
		return nil, fmt.Errorf("error parsing config JSON: %w", err)
	}

	// Extract files from config
	configFiles, ok := configData["files"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("files not found in config or not an array")
	}

	// Create a map of file info for quick lookup
	fileInfo := make(map[string]string) // map[filename]fileID
	for _, f := range configFiles {
		file, ok := f.(map[string]interface{})
		if !ok {
			continue
		}
		filename, filenameOk := file["filename"].(string)
		id, idOk := file["id"].(string)
		if filenameOk && idOk {
			fileInfo[filename] = id
		}
	}

	// Read only the files listed in the config
	files := make(map[string][]byte)
	for filename, fileID := range fileInfo {
		// Construct the full path to the file
		filePath := filepath.Join(rolesTemplatesDir, filename+".j2")

		// Check if the file exists
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			log.Printf("Warning: File %s listed in config but not found in templates directory", filename)
			continue
		}

		// Read the file content
		content, err := os.ReadFile(filePath)
		if err != nil {
			return nil, fmt.Errorf("error reading template file %s: %w", filePath, err)
		}

		// Add the file to the map
		files[fileID] = content
	}

	// Combine variables and secrets into a single map
	for k, _ := range secrets {
		variables[k] = "<encrypted>"
	}

	// Parse config bytes into AppConfig
	appConfig, err := model.ParseAppConfig(configBytes)
	if err != nil {
		return nil, fmt.Errorf("error parsing config: %w", err)
	}

	// Create and return the app data
	return &model.App{
		ID:        appID,
		Config:    appConfig,
		Variables: variables,
		Files:     files,
	}, nil
}

// NewGetAppQueryHandler creates a new GetAppQueryHandler
func NewGetAppQueryHandler(ansibleAppsRolesPath, ansibleAppsRolesCurrentVersion string) *GetAppQueryHandler {
	return &GetAppQueryHandler{
		AnsibleAppsRolesPath:           ansibleAppsRolesPath,
		AnsibleAppsRolesCurrentVersion: ansibleAppsRolesCurrentVersion,
	}
}
