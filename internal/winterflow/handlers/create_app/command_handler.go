package create_app

import (
	"fmt"
	"os"
	"path/filepath"
	"winterflow-agent/internal/config"
	"winterflow-agent/internal/winterflow/grpc/pb"
	log "winterflow-agent/pkg/log"
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
	configDir := filepath.Join(config.GetAnsibleAppsPath(), "configs")
	rolesDir := filepath.Join(config.GetAnsibleAppsPath(), "roles", appID)
	inventoryDir := filepath.Join(config.GetAnsibleAppsPath(), "inventory", appID)

	// Create directories if they don't exist
	if err := os.MkdirAll(configDir, 0755); err != nil {
		log.Printf("Error creating config directory: %v", err)
		responseCode = pb.ResponseCode_RESPONSE_CODE_SERVER_ERROR
		responseMessage = fmt.Sprintf("Error creating config directory: %v", err)
	}

	if err := os.MkdirAll(rolesDir, 0755); err != nil {
		log.Printf("Error creating roles directory: %v", err)
		responseCode = pb.ResponseCode_RESPONSE_CODE_SERVER_ERROR
		responseMessage = fmt.Sprintf("Error creating roles directory: %v", err)
	}

	if err := os.MkdirAll(inventoryDir, 0755); err != nil {
		log.Printf("Error creating inventory directory: %v", err)
		responseCode = pb.ResponseCode_RESPONSE_CODE_SERVER_ERROR
		responseMessage = fmt.Sprintf("Error creating inventory directory: %v", err)
	}

	// Create the files
	if responseCode == pb.ResponseCode_RESPONSE_CODE_SUCCESS {
		// Create config file
		configFile := filepath.Join(configDir, appID+".json")
		if err := os.WriteFile(configFile, cmd.Request.App.Config, 0644); err != nil {
			log.Printf("Error creating config file: %v", err)
			responseCode = pb.ResponseCode_RESPONSE_CODE_SERVER_ERROR
			responseMessage = fmt.Sprintf("Error creating config file: %v", err)
		}

		// Create role files
		for _, file := range cmd.Request.App.Files {
			roleFile := filepath.Join(rolesDir, file.Name+".j2")
			if err := os.WriteFile(roleFile, file.Content, 0644); err != nil {
				log.Printf("Error creating role file %s: %v", file.Name, err)
				responseCode = pb.ResponseCode_RESPONSE_CODE_SERVER_ERROR
				responseMessage = fmt.Sprintf("Error creating role file %s: %v", file.Name, err)
				break
			}
		}

		// Create inventory files
		varsFile := filepath.Join(inventoryDir, "vars.yml")
		if err := os.WriteFile(varsFile, cmd.Request.App.Variables, 0644); err != nil {
			log.Printf("Error creating vars file: %v", err)
			responseCode = pb.ResponseCode_RESPONSE_CODE_SERVER_ERROR
			responseMessage = fmt.Sprintf("Error creating vars file: %v", err)
		}

		secretsFile := filepath.Join(inventoryDir, "secrets.yml")
		if err := os.WriteFile(secretsFile, cmd.Request.App.Secrets, 0644); err != nil {
			log.Printf("Error creating secrets file: %v", err)
			responseCode = pb.ResponseCode_RESPONSE_CODE_SERVER_ERROR
			responseMessage = fmt.Sprintf("Error creating secrets file: %v", err)
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
