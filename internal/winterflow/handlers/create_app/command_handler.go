package create_app

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"winterflow-agent/internal/config"
	"winterflow-agent/internal/winterflow/grpc/pb"
	log "winterflow-agent/pkg/log"
	"winterflow-agent/pkg/yaml"
)

// CreateAppHandler handles the CreateAppCommand
type CreateAppHandler struct{}

// Handle executes the CreateAppCommand
func (h *CreateAppHandler) Handle(cmd CreateAppCommand) error {
	log.Printf("Processing create app request for app ID: %s", cmd.Request.App.AppId)

	// Create the directory structure and files
	appID := cmd.Request.App.AppId
	var responseCode pb.ResponseCode = pb.ResponseCode_RESPONSE_CODE_SUCCESS
	var responseMessage string = "App created successfully"

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

	// Create the files
	if responseCode == pb.ResponseCode_RESPONSE_CODE_SUCCESS {
		// Create config file
		roleConfigFile := filepath.Join(rolesDir, "config.json")

		// Store config.json in roles/{APP_ID}/config.json
		if err := os.WriteFile(roleConfigFile, cmd.Request.App.Config, 0644); err != nil {
			log.Error("Error creating role config file: %v", err)
			responseCode = pb.ResponseCode_RESPONSE_CODE_SERVER_ERROR
			responseMessage = fmt.Sprintf("Error creating role config file: %v", err)
		}

		// Create role files
		for _, file := range cmd.Request.App.Files {
			roleFile := filepath.Join(rolesTemplatesDir, file.Name+".j2")
			if err := os.WriteFile(roleFile, file.Content, 0644); err != nil {
				log.Error("Error creating role file %s: %v", file.Name, err)
				responseCode = pb.ResponseCode_RESPONSE_CODE_SERVER_ERROR
				responseMessage = fmt.Sprintf("Error creating role file %s: %v", file.Name, err)
				break
			}
		}

		// Create defaults/main.yml with empty values based on config variables
		var configData map[string]interface{}
		if err := json.Unmarshal(cmd.Request.App.Config, &configData); err != nil {
			log.Error("Error parsing config JSON: %v", err)
			responseCode = pb.ResponseCode_RESPONSE_CODE_SERVER_ERROR
			responseMessage = fmt.Sprintf("Error parsing config JSON: %v", err)
		} else {
			// Extract variables from config
			variables, ok := configData["variables"].([]interface{})
			if ok {
				// Create empty values map
				emptyValues := make(map[string]string)
				for _, v := range variables {
					variable, ok := v.(map[string]interface{})
					if !ok {
						continue
					}
					name, ok := variable["name"].(string)
					if ok {
						emptyValues[name] = ""
					}
				}

				// Convert to JSON and then to YAML
				emptyValuesJSON, err := json.Marshal(emptyValues)
				if err != nil {
					log.Printf("Error marshaling empty values to JSON: %v", err)
					responseCode = pb.ResponseCode_RESPONSE_CODE_SERVER_ERROR
					responseMessage = fmt.Sprintf("Error marshaling empty values to JSON: %v", err)
				} else {
					defaultsYAML, err := yaml.JSONToYAML(emptyValuesJSON)
					if err != nil {
						log.Error("Error converting defaults to YAML: %v", err)
						responseCode = pb.ResponseCode_RESPONSE_CODE_SERVER_ERROR
						responseMessage = fmt.Sprintf("Error converting defaults to YAML: %v", err)
					} else {
						defaultsFile := filepath.Join(rolesDefaultsDir, "main.yml")
						if err := os.WriteFile(defaultsFile, defaultsYAML, 0644); err != nil {
							log.Error("Error creating defaults file: %v", err)
							responseCode = pb.ResponseCode_RESPONSE_CODE_SERVER_ERROR
							responseMessage = fmt.Sprintf("Error creating defaults file: %v", err)
						}
					}
				}
			}
		}

		// Process variables
		varsJSON, err := ReplaceIDsWithNames(cmd.Request.App.Config, cmd.Request.App.Variables)
		if err != nil {
			log.Error("Error replacing IDs with NAMEs: %v", err)
			responseCode = pb.ResponseCode_RESPONSE_CODE_SERVER_ERROR
			responseMessage = fmt.Sprintf("Error replacing IDs with NAMEs: %v", err)
		}

		// Convert Variables from JSON to YAML using variable names from config
		varsYAML, err := yaml.JSONToYAML(varsJSON)
		if err != nil {
			log.Error("Error converting variables to YAML: %v", err)
			responseCode = pb.ResponseCode_RESPONSE_CODE_SERVER_ERROR
			responseMessage = fmt.Sprintf("Error converting variables to YAML: %v", err)
		} else {
			// Create vars.yml
			newVarsFile := filepath.Join(rolesVarsDir, "vars.yml")
			if err := os.WriteFile(newVarsFile, varsYAML, 0644); err != nil {
				log.Error("Error creating vars file: %v", err)
				responseCode = pb.ResponseCode_RESPONSE_CODE_SERVER_ERROR
				responseMessage = fmt.Sprintf("Error creating vars file: %v", err)
			}
		}

		// Process secrets
		secretsJSON, err := ReplaceIDsWithNames(cmd.Request.App.Config, cmd.Request.App.Secrets)
		if err != nil {
			log.Error("Error replacing IDs with NAMEs: %v", err)
			responseCode = pb.ResponseCode_RESPONSE_CODE_SERVER_ERROR
			responseMessage = fmt.Sprintf("Error replacing IDs with NAMEs: %v", err)
		}

		// Convert Secrets from JSON to YAML using variable names from config
		secretsYAML, err := yaml.JSONToYAML(secretsJSON)
		if err != nil {
			log.Error("Error converting secrets to YAML: %v", err)
			responseCode = pb.ResponseCode_RESPONSE_CODE_SERVER_ERROR
			responseMessage = fmt.Sprintf("Error converting secrets to YAML: %v", err)
		} else {
			// Create secrets.yml
			newSecretsFile := filepath.Join(rolesVarsDir, "secrets.yml")
			if err := os.WriteFile(newSecretsFile, secretsYAML, 0644); err != nil {
				log.Error("Error creating secrets file: %v", err)
				responseCode = pb.ResponseCode_RESPONSE_CODE_SERVER_ERROR
				responseMessage = fmt.Sprintf("Error creating secrets file: %v", err)
			}
		}
	}

	// Return error if there was a problem
	if responseCode != pb.ResponseCode_RESPONSE_CODE_SUCCESS {
		return fmt.Errorf(responseMessage)
	}

	return nil
}

// CreateAppResult represents the result of creating an app
type CreateAppResult struct {
	ResponseCode    pb.ResponseCode
	ResponseMessage string
	App             *pb.AppV1
}

// NewCreateAppHandler creates a new CreateAppHandler
func NewCreateAppHandler() *CreateAppHandler {
	return &CreateAppHandler{}
}
