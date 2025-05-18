package save_app

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"winterflow-agent/internal/config"
	"winterflow-agent/internal/winterflow/grpc/pb"
	"winterflow-agent/internal/winterflow/models"
	log "winterflow-agent/pkg/log"
	"winterflow-agent/pkg/yaml"
)

// SaveAppHandler handles the SaveAppCommand
type SaveAppHandler struct{}

// Handle executes the SaveAppCommand
func (h *SaveAppHandler) Handle(cmd SaveAppCommand) error {
	log.Printf("Processing save app request for app ID: %s", cmd.Request.App.AppId)

	// Validate the app configuration
	appConfig, err := models.ParseAppConfig(cmd.Request.App.Config)
	if err != nil {
		log.Error("Error parsing app config: %v", err)
		return fmt.Errorf("error parsing app config: %v", err)
	}

	// Parse variables and secrets
	variables := models.ParseVariableMapFromProto(cmd.Request.App.Variables)
	secrets := models.ParseVariableMapFromProto(cmd.Request.App.Secrets)

	// Create the directory structure and files
	appID := cmd.Request.App.AppId
	var responseCode pb.ResponseCode = pb.ResponseCode_RESPONSE_CODE_SUCCESS
	var responseMessage string = "App saved successfully"

	// Create the required directories
	rolesDir := filepath.Join(config.GetAnsibleAppsRolesPath(), appID)
	rolesDefaultsDir := filepath.Join(rolesDir, "defaults")
	rolesVarsDir := filepath.Join(rolesDir, "vars")
	rolesTemplatesDir := filepath.Join(rolesDir, "templates")

	// Create directories if they don't exist
	if err := os.MkdirAll(rolesDir, 0755); err != nil {
		log.Error("Error creating roles directory: %v", err)
		responseCode = pb.ResponseCode_RESPONSE_CODE_SERVER_ERROR
		responseMessage = fmt.Sprintf("Error creating roles directory: %v", err)
	}

	if err := os.MkdirAll(rolesDefaultsDir, 0755); err != nil {
		log.Error("Error creating roles defaults directory: %v", err)
		responseCode = pb.ResponseCode_RESPONSE_CODE_SERVER_ERROR
		responseMessage = fmt.Sprintf("Error creating roles defaults directory: %v", err)
	}

	if err := os.MkdirAll(rolesVarsDir, 0755); err != nil {
		log.Error("Error creating roles vars directory: %v", err)
		responseCode = pb.ResponseCode_RESPONSE_CODE_SERVER_ERROR
		responseMessage = fmt.Sprintf("Error creating roles vars directory: %v", err)
	}

	if err := os.MkdirAll(rolesTemplatesDir, 0755); err != nil {
		log.Error("Error creating roles templates directory: %v", err)
		responseCode = pb.ResponseCode_RESPONSE_CODE_SERVER_ERROR
		responseMessage = fmt.Sprintf("Error creating roles templates directory: %v", err)
	}

	// Process files, variables, and secrets
	if responseCode == pb.ResponseCode_RESPONSE_CODE_SUCCESS {
		// Create config file
		roleConfigFile := filepath.Join(rolesDir, "config.json")

		// Store config.json in roles/{APP_ID}/config.json
		if err := os.WriteFile(roleConfigFile, cmd.Request.App.Config, 0644); err != nil {
			log.Error("Error creating role config file: %v", err)
			responseCode = pb.ResponseCode_RESPONSE_CODE_SERVER_ERROR
			responseMessage = fmt.Sprintf("Error creating role config file: %v", err)
		}

		// Handle template files
		if err := h.handleTemplateFiles(rolesTemplatesDir, appConfig, cmd.Request.App.Files); err != nil {
			log.Error("Error handling template files: %v", err)
			responseCode = pb.ResponseCode_RESPONSE_CODE_SERVER_ERROR
			responseMessage = fmt.Sprintf("Error handling template files: %v", err)
		}

		// Create defaults/main.yml with empty values based on config variables
		if err := h.createDefaultsFile(rolesDefaultsDir, appConfig); err != nil {
			log.Error("Error creating defaults file: %v", err)
			responseCode = pb.ResponseCode_RESPONSE_CODE_SERVER_ERROR
			responseMessage = fmt.Sprintf("Error creating defaults file: %v", err)
		}

		// Process variables
		if err := h.processVariables(rolesVarsDir, appConfig, variables); err != nil {
			log.Error("Error processing variables: %v", err)
			responseCode = pb.ResponseCode_RESPONSE_CODE_SERVER_ERROR
			responseMessage = fmt.Sprintf("Error processing variables: %v", err)
		}

		// Process secrets
		if err := h.processSecrets(rolesVarsDir, appConfig, secrets); err != nil {
			log.Error("Error processing secrets: %v", err)
			responseCode = pb.ResponseCode_RESPONSE_CODE_SERVER_ERROR
			responseMessage = fmt.Sprintf("Error processing secrets: %v", err)
		}
	}

	// Return error if there was a problem
	if responseCode != pb.ResponseCode_RESPONSE_CODE_SUCCESS {
		return fmt.Errorf(responseMessage)
	}

	return nil
}

// handleTemplateFiles handles the template files for the app
// It creates new files, updates existing files, and deletes files that are no longer in the config
func (h *SaveAppHandler) handleTemplateFiles(templatesDir string, appConfig *models.AppConfig, files []*pb.AppFileV1) error {
	// Create a map of filenames from the request for creating/updating files
	requestFiles := make(map[string]bool)
	for _, file := range files {
		requestFiles[file.Id] = true
	}

	// Create a map of filenames from the appConfig for checking which files to delete
	configFiles := make(map[string]bool)
	for _, file := range appConfig.Files {
		configFiles[file.Filename] = true
	}

	// Get existing template files
	existingFiles, err := filepath.Glob(filepath.Join(templatesDir, "*.j2"))
	if err != nil {
		return fmt.Errorf("error getting existing template files: %w", err)
	}

	// Delete files that are no longer in the config
	for _, existingFile := range existingFiles {
		filename := filepath.Base(existingFile)
		// Remove the .j2 extension
		filename = filename[:len(filename)-3]
		if !configFiles[filename] {
			if err := os.Remove(existingFile); err != nil {
				return fmt.Errorf("error removing file %s: %w", existingFile, err)
			}
			log.Debug("Deleted file: %s", existingFile)
		}
	}

	// Create or update files from the request
	for _, file := range files {
		templateFile := filepath.Join(templatesDir, file.Id+".j2")
		if err := os.WriteFile(templateFile, file.Content, 0644); err != nil {
			return fmt.Errorf("error creating/updating template file %s: %w", file.Id, err)
		}
		log.Printf("Created/updated file: %s", templateFile)
	}

	return nil
}

// createDefaultsFile creates the defaults/main.yml file with empty values based on config variables
func (h *SaveAppHandler) createDefaultsFile(defaultsDir string, appConfig *models.AppConfig) error {
	// Create empty values map
	emptyValues := make(map[string]string)
	for _, v := range appConfig.Variables {
		emptyValues[v.Name] = ""
	}

	// Convert to JSON and then to YAML
	emptyValuesJSON, err := json.Marshal(emptyValues)
	if err != nil {
		return fmt.Errorf("error marshaling empty values to JSON: %w", err)
	}

	defaultsYAML, err := yaml.JSONToYAML(emptyValuesJSON)
	if err != nil {
		return fmt.Errorf("error converting defaults to YAML: %w", err)
	}

	defaultsFile := filepath.Join(defaultsDir, "main.yml")
	if err := os.WriteFile(defaultsFile, defaultsYAML, 0644); err != nil {
		return fmt.Errorf("error creating defaults file: %w", err)
	}

	return nil
}

// processVariables processes the variables for the app
func (h *SaveAppHandler) processVariables(varsDir string, appConfig *models.AppConfig, variables models.VariableMap) error {
	// Map variable IDs to names using the appConfig
	idToName := make(map[string]string)
	// Create a set of variable IDs from appConfig for checking which variables to keep
	configVarIDs := make(map[string]bool)
	configNameToID := make(map[string]string) // Map variable names to IDs
	for _, v := range appConfig.Variables {
		idToName[v.ID] = v.Name
		configVarIDs[v.ID] = true
		configNameToID[v.Name] = v.ID
	}

	// Check if vars.yml exists and read it
	varsFile := filepath.Join(varsDir, "vars.yml")
	existingNamedVariables := make(map[string]string)
	if _, err := os.Stat(varsFile); err == nil {
		// File exists, read it
		varsYAML, err := os.ReadFile(varsFile)
		if err != nil {
			return fmt.Errorf("error reading existing vars file: %w", err)
		}

		// Parse YAML manually since it's a simple key-value format
		lines := strings.Split(string(varsYAML), "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line == "" || line == "{}" {
				continue
			}
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				key := strings.TrimSpace(parts[0])
				value := strings.TrimSpace(parts[1])
				// Remove any trailing characters like '%'
				if len(value) > 0 && value[len(value)-1] == '%' {
					value = value[:len(value)-1]
				}
				existingNamedVariables[key] = value
			}
		}

		// Keep only variables that are in the appConfig
		for name := range existingNamedVariables {
			if _, exists := configNameToID[name]; !exists {
				// Variable not in appConfig, remove it
				delete(existingNamedVariables, name)
			}
		}
	}

	// Replace IDs with names and only include variables that are in the appConfig
	for id, value := range variables {
		// Only process variables that are in the appConfig
		if configVarIDs[id] {
			name, ok := idToName[id]
			if ok {
				existingNamedVariables[name] = value
			} else {
				// Keep original ID if name not found
				existingNamedVariables[id] = value
			}
		}
	}

	// Convert to JSON and then to YAML
	varsJSON, err := json.Marshal(existingNamedVariables)
	if err != nil {
		return fmt.Errorf("error marshaling variables to JSON: %w", err)
	}

	// Convert Variables from JSON to YAML
	varsYAML, err := yaml.JSONToYAML(varsJSON)
	if err != nil {
		return fmt.Errorf("error converting variables to YAML: %w", err)
	}

	// Create vars.yml
	if err := os.WriteFile(varsFile, varsYAML, 0644); err != nil {
		return fmt.Errorf("error creating vars file: %w", err)
	}

	return nil
}

// processSecrets processes the secrets for the app
func (h *SaveAppHandler) processSecrets(varsDir string, appConfig *models.AppConfig, secrets models.VariableMap) error {
	// Map variable IDs to names using the appConfig
	idToName := make(map[string]string)
	// Create a set of variable IDs from appConfig for checking which secrets to keep
	configVarIDs := make(map[string]bool)
	configNameToID := make(map[string]string) // Map variable names to IDs
	for _, v := range appConfig.Variables {
		idToName[v.ID] = v.Name
		configVarIDs[v.ID] = true
		configNameToID[v.Name] = v.ID
	}

	// Check if secrets.yml exists and read it
	secretsFile := filepath.Join(varsDir, "secrets.yml")
	existingNamedSecrets := make(map[string]string)
	if _, err := os.Stat(secretsFile); err == nil {
		// File exists, read it
		secretsYAML, err := os.ReadFile(secretsFile)
		if err != nil {
			return fmt.Errorf("error reading existing secrets file: %w", err)
		}

		// Parse YAML manually since it's a simple key-value format
		lines := strings.Split(string(secretsYAML), "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line == "" || line == "{}" {
				continue
			}
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				key := strings.TrimSpace(parts[0])
				value := strings.TrimSpace(parts[1])
				// Remove any trailing characters like '%'
				if len(value) > 0 && value[len(value)-1] == '%' {
					value = value[:len(value)-1]
				}
				existingNamedSecrets[key] = value
			}
		}

		// Keep only secrets that are in the appConfig
		for name := range existingNamedSecrets {
			if _, exists := configNameToID[name]; !exists {
				// Secret not in appConfig, remove it
				delete(existingNamedSecrets, name)
			}
		}
	}

	// Replace IDs with names and only include secrets that are in the appConfig
	for id, value := range secrets {
		// Only process secrets that are in the appConfig
		if configVarIDs[id] {
			name, ok := idToName[id]
			if ok {
				existingNamedSecrets[name] = value
			} else {
				// Keep original ID if name not found
				existingNamedSecrets[id] = value
			}
		}
	}

	// Convert to JSON and then to YAML
	secretsJSON, err := json.Marshal(existingNamedSecrets)
	if err != nil {
		return fmt.Errorf("error marshaling secrets to JSON: %w", err)
	}

	// Convert Secrets from JSON to YAML
	secretsYAML, err := yaml.JSONToYAML(secretsJSON)
	if err != nil {
		return fmt.Errorf("error converting secrets to YAML: %w", err)
	}

	// Create secrets.yml
	if err := os.WriteFile(secretsFile, secretsYAML, 0644); err != nil {
		return fmt.Errorf("error creating secrets file: %w", err)
	}

	return nil
}

// SaveAppResult represents the result of creating an app
type SaveAppResult struct {
	ResponseCode    pb.ResponseCode
	ResponseMessage string
	App             *pb.AppV1
}

// NewSaveAppHandler creates a new SaveAppHandler
func NewSaveAppHandler() *SaveAppHandler {
	return &SaveAppHandler{}
}
