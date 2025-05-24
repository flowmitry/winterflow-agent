package get_app

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"winterflow-agent/internal/winterflow/grpc/pb"
	log "winterflow-agent/pkg/log"
)

// GetAppQueryHandler handles the GetAppQuery
type GetAppQueryHandler struct {
	AnsibleAppsRolesPath           string
	AnsibleAppsRolesCurrentVersion string
}

// Handle executes the GetAppQuery and returns the result
func (h *GetAppQueryHandler) Handle(query GetAppQuery) (*pb.AppV1, error) {
	log.Printf("Processing get app request for app ID: %s", query.Request.AppId)

	// Get the app ID from the request
	appID := query.Request.AppId

	// Determine the app version to use
	var versionDir string
	if query.Request.AppVersion > 0 {
		versionDir = fmt.Sprintf("%d", query.Request.AppVersion)
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

	// Convert variables JSON to AppVarV1 slice
	variables, err := convertJSONToAppVars(varsJSON)
	if err != nil {
		return nil, fmt.Errorf("error converting variables JSON to AppVarV1: %w", err)
	}

	// Convert secrets from YAML to JSON with "id": "value" format
	secretsJSON, err := convertYAMLToIDValueJSON(configBytes, secretsBytes)
	if err != nil {
		return nil, fmt.Errorf("error converting secrets to JSON: %w", err)
	}

	// Convert secrets JSON to AppVarV1 slice
	secrets, err := convertJSONToAppVars(secretsJSON)
	if err != nil {
		return nil, fmt.Errorf("error converting secrets JSON to AppVarV1: %w", err)
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
	var files []*pb.AppFileV1
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

		// Create an AppFileV1 for the file
		file := &pb.AppFileV1{
			Id:      fileID,
			Content: content,
		}
		files = append(files, file)
	}

	// Create and return the app data
	return &pb.AppV1{
		AppId:     appID,
		Config:    configBytes,
		Variables: variables,
		Secrets:   secrets,
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
