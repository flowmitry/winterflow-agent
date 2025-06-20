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
	log.Printf("Processing save app request for app ID: %s", cmd.App.ID)

	// Get the app configuration from the command
	appConfig := cmd.App.Config

	// Get variables from the command
	variables := cmd.App.Variables

	// Create the directory structure and files
	appID := cmd.App.ID
	var success bool = true
	var responseMessage string = "App saved successfully"

	// Create the required directories
	rolesDir := filepath.Join(h.AnsibleAppsRolesPath, appID)
	versionDir := filepath.Join(rolesDir, h.AnsibleAppsRolesCurrentVersion)
	rolesDefaultsDir := filepath.Join(versionDir, "defaults")
	rolesVarsDir := filepath.Join(versionDir, "vars")
	rolesVarsFile := filepath.Join(rolesVarsDir, "vars.yml")
	rolesSecretsFile := filepath.Join(rolesVarsDir, "secrets.yml")
	rolesTemplatesDir := filepath.Join(versionDir, "templates")

	// Create directories if they don't exist
	if err := os.MkdirAll(versionDir, 0755); err != nil {
		log.Error("Error creating roles directory: %v", err)
		success = false
		responseMessage = fmt.Sprintf("Error creating roles directory: %v", err)
	}

	if err := os.MkdirAll(rolesDefaultsDir, 0755); err != nil {
		log.Error("Error creating roles defaults directory: %v", err)
		success = false
		responseMessage = fmt.Sprintf("Error creating roles defaults directory: %v", err)
	}

	if err := os.MkdirAll(rolesVarsDir, 0755); err != nil {
		log.Error("Error creating roles vars directory: %v", err)
		success = false
		responseMessage = fmt.Sprintf("Error creating roles vars directory: %v", err)
	}

	if err := os.MkdirAll(rolesTemplatesDir, 0755); err != nil {
		log.Error("Error creating roles templates directory: %v", err)
		success = false
		responseMessage = fmt.Sprintf("Error creating roles templates directory: %v", err)
	}

	// Process files, variables, and secrets
	if success {
		// Create config file
		roleConfigFile := filepath.Join(versionDir, "config.json")

		// Convert AppConfig to JSON
		configBytes, err := json.Marshal(appConfig)
		if err != nil {
			log.Error("Error marshaling app config: %v", err)
			success = false
			responseMessage = fmt.Sprintf("Error marshaling app config: %v", err)
		}

		// Store config.json in app_roles/{APP_ID}/config.json
		if success {
			if err := os.WriteFile(roleConfigFile, configBytes, 0644); err != nil {
				log.Error("Error creating role config file: %v", err)
				success = false
				responseMessage = fmt.Sprintf("Error creating role config file: %v", err)
			}
		}

		// Handle template files
		if success {
			if err := h.handleTemplateFiles(rolesTemplatesDir, appConfig, cmd.App.Files); err != nil {
				log.Error("Error handling template files: %v", err)
				success = false
				responseMessage = fmt.Sprintf("Error handling template files: %v", err)
			}
		}

		// Create defaults/main.yml with empty values based on config variables
		if success {
			if err := h.createDefaultsFile(rolesDefaultsDir, appConfig); err != nil {
				log.Error("Error creating defaults file: %v", err)
				success = false
				responseMessage = fmt.Sprintf("Error creating defaults file: %v", err)
			}
		}

		// Process variables
		if success {
			if err := h.processVariables(rolesVarsFile, appConfig, variables); err != nil {
				log.Error("Error processing variables: %v", err)
				success = false
				responseMessage = fmt.Sprintf("Error processing variables: %v", err)
			}
		}

		// Process secrets
		if success {
			if err := h.processSecrets(rolesSecretsFile, appConfig, variables); err != nil {
				log.Error("Error processing secrets: %v", err)
				success = false
				responseMessage = fmt.Sprintf("Error processing secrets: %v", err)
			}
		}
	}

	// Return error if there was a problem
	if !success {
		return fmt.Errorf(responseMessage)
	}

	return nil
}

// handleTemplateFiles handles the template files for the app
// It creates new files, updates existing files, and deletes files that are no longer in the config
func (h *SaveAppHandler) handleTemplateFiles(templatesDir string, appConfig *model.AppConfig, files model.FilesMap) error {
	// Create a map of filenames from the request for creating/updating files
	requestFiles := make(map[string]bool)
	for fileID := range files {
		requestFiles[fileID] = true
	}

	// Create a map of filenames from the appConfig for checking which files to delete
	configFiles := make(map[string]bool)
	// Create a map to lookup filenames by ID
	idToFilename := make(map[string]string)
	for _, file := range appConfig.Files {
		configFiles[file.Filename] = true
		idToFilename[file.ID] = file.Filename
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
	for fileID, content := range files {
		// Get the filename from the ID using the mapping
		filename, ok := idToFilename[fileID]
		if !ok {
			// If filename not found, fall back to using ID
			filename = fileID
			log.Printf("Warning: No filename found for ID %s, using ID as filename", fileID)
		}

		templateFile := filepath.Join(templatesDir, filename+".j2")
		if err := os.WriteFile(templateFile, []byte(content), 0644); err != nil {
			return fmt.Errorf("error creating/updating template file %s: %w", filename, err)
		}
		log.Printf("Created/updated file: %s", templateFile)
	}

	return nil
}

// createDefaultsFile creates the defaults/main.yml file with empty values based on config variables
func (h *SaveAppHandler) createDefaultsFile(defaultsDir string, appConfig *model.AppConfig) error {
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

// processMap is a helper function that processes variables or secrets for the app
func (h *SaveAppHandler) processMap(filePath string, appConfig *model.AppConfig, variableMap model.VariableMap, decrypt bool) error {
	// Map variable IDs to names using the appConfig
	idToName := make(map[string]string)
	// Create a set of variable IDs from appConfig for checking which variables/secrets to keep
	configVarIDs := make(map[string]bool)
	configNameToID := make(map[string]string) // Map variable names to IDs
	for _, v := range appConfig.Variables {
		idToName[v.ID] = v.Name
		configVarIDs[v.ID] = true
		configNameToID[v.Name] = v.ID
	}

	// Check if file exists and read it
	existingNamedValues := make(map[string]string)
	if _, err := os.Stat(filePath); err == nil {
		// File exists, read it
		yamlData, err := os.ReadFile(filePath)
		if err != nil {
			return fmt.Errorf("error reading existing %s file: %w", filePath, err)
		}

		// Parse YAML using pkg/yaml
		if err := yaml.UnmarshalYAML(yamlData, &existingNamedValues); err != nil {
			return fmt.Errorf("error parsing existing %s file: %w", filePath, err)
		}

		// Keep only values that are in the appConfig
		for name := range existingNamedValues {
			if _, exists := configNameToID[name]; !exists {
				// Value not in appConfig, remove it
				delete(existingNamedValues, name)
			}
		}
	}

	// Replace IDs with names and only include values that are in the appConfig
	for id, value := range variableMap {
		// Only process values that are in the appConfig
		if configVarIDs[id] {
			// If the value is "<encrypted>", use the current value from the secrets file
			if value == "<encrypted>" && decrypt {
				name, ok := idToName[id]
				if ok && existingNamedValues[name] != "" {
					// Skip this value as we'll keep the existing one
					continue
				}
			}

			// Decrypt the value if needed
			if decrypt && h.PrivateKeyPath != "" {
				decryptedValue, err := certs.DecryptWithPrivateKey(h.PrivateKeyPath, value)
				if err != nil {
					log.Error("Error decrypting value for ID %s: %v", id, err)
					// Continue with the original value if decryption fails
				} else {
					value = decryptedValue
				}
			}

			name, ok := idToName[id]
			if ok {
				existingNamedValues[name] = value
			} else {
				// Keep original ID if name not found
				existingNamedValues[id] = value
			}
		}
	}

	// If the map is empty, create an empty file
	if len(existingNamedValues) == 0 {
		// Create empty file
		if err := os.WriteFile(filePath, []byte{}, 0644); err != nil {
			return fmt.Errorf("error creating empty %s file: %w", filePath, err)
		}
	} else {
		// Convert to JSON and then to YAML
		valuesJSON, err := json.Marshal(existingNamedValues)
		if err != nil {
			return fmt.Errorf("error marshaling values to JSON: %w", err)
		}

		// Convert from JSON to YAML
		valuesYAML, err := yaml.JSONToYAML(valuesJSON)
		if err != nil {
			return fmt.Errorf("error converting values to YAML: %w", err)
		}

		// Create file
		if err := os.WriteFile(filePath, valuesYAML, 0644); err != nil {
			return fmt.Errorf("error creating %s file: %w", filePath, err)
		}
	}

	return nil
}

// processVariables processes the variables for the app
func (h *SaveAppHandler) processVariables(varsFile string, appConfig *model.AppConfig, variables model.VariableMap) error {
	// Filter variables to only include non-secrets
	nonSecretVars := make(model.VariableMap)
	for _, v := range appConfig.Variables {
		if !v.IsSecret {
			if value, exists := variables[v.ID]; exists {
				nonSecretVars[v.ID] = value
			}
		}
	}
	return h.processMap(varsFile, appConfig, nonSecretVars, false)
}

// processSecrets processes the secrets for the app
func (h *SaveAppHandler) processSecrets(secretsFile string, appConfig *model.AppConfig, variables model.VariableMap) error {
	// Filter variables to only include secrets
	secretVars := make(model.VariableMap)
	for _, v := range appConfig.Variables {
		if v.IsSecret {
			if value, exists := variables[v.ID]; exists {
				secretVars[v.ID] = value
			}
		}
	}
	return h.processMap(secretsFile, appConfig, secretVars, true)
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
