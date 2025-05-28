package get_apps_status

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"winterflow-agent/internal/winterflow/ansible"
	"winterflow-agent/internal/winterflow/grpc/pb"
	log "winterflow-agent/pkg/log"
)

// GetAppsStatusQueryHandler handles the GetAppsStatusQuery
type GetAppsStatusQueryHandler struct {
	ansible ansible.Repository
}

// Handle executes the GetAppsStatusQuery and returns the result
func (h *GetAppsStatusQueryHandler) Handle(query GetAppsStatusQuery) ([]*pb.AppStatusV1, error) {
	log.Printf("Processing get apps status request")

	// Create a temporary directory for app status files
	tempAppsStatusDir, err := os.MkdirTemp("", "apps_status_*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temporary directory: %w", err)
	}
	defer os.RemoveAll(tempAppsStatusDir)

	result := h.ansible.GenerateAppsStatus(tempAppsStatusDir)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to run get_apps_status playbook: %w", result.Error)
	}

	// Read the status files from the output directory
	statusFiles, err := os.ReadDir(tempAppsStatusDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read status output directory: %w", err)
	}

	var appStatuses []*pb.AppStatusV1

	// Process each status file
	for _, file := range statusFiles {
		if file.IsDir() || !strings.HasSuffix(file.Name(), ".json") {
			continue
		}

		// Extract app ID from filename (remove .json extension)
		appID := strings.TrimSuffix(file.Name(), ".json")

		// Read the status file
		statusFilePath := filepath.Join(tempAppsStatusDir, file.Name())
		statusData, err := os.ReadFile(statusFilePath)
		if err != nil {
			log.Printf("Error reading status file %s: %v", statusFilePath, err)
			continue
		}

		// Parse the JSON data
		var containers []map[string]interface{}
		if err := json.Unmarshal(statusData, &containers); err != nil {
			log.Printf("Error parsing JSON from status file %s: %v", statusFilePath, err)
			continue
		}

		// Determine the overall status of the app based on container statuses
		statusCode := determineAppStatus(containers)

		// Create an AppStatusV1 for this app
		appStatus := &pb.AppStatusV1{
			AppId:      appID,
			StatusCode: statusCode,
		}

		appStatuses = append(appStatuses, appStatus)
	}

	return appStatuses, nil
}

// determineAppStatus determines the overall status of an app based on its container statuses
func determineAppStatus(containers []map[string]interface{}) pb.AppStatusCode {
	if len(containers) == 0 {
		return pb.AppStatusCode_STATUS_CODE_STOPPED
	}

	// Count containers in each state
	var active, idle, stopped, restarting, problematic int

	for _, container := range containers {
		state, ok := container["State"].(string)
		if !ok {
			continue
		}

		status, ok := container["Status"].(string)
		if !ok {
			status = ""
		}

		exitCode, ok := container["ExitCode"].(int)
		if !ok {
			exitCode = 0
		}

		// Map Docker container states to app status codes
		switch {
		case state == "active":
			if strings.Contains(status, "unhealthy") {
				problematic++
			} else {
				active++
			}
		case state == "created" || state == "paused":
			idle++
		case state == "exited":
		case state == "removing":
			stopped++
		case state == "dead":
			problematic++
		case state == "restarting":
			if exitCode != 0 {
				problematic++
			} else {
				restarting++
			}
		default:
			problematic++
		}
	}

	// Determine overall status based on container counts
	if problematic > 0 {
		return pb.AppStatusCode_STATUS_CODE_PROBLEMATIC
	}
	if restarting > 0 {
		return pb.AppStatusCode_STATUS_CODE_RESTARTING
	}
	if active > 0 && stopped == 0 && idle == 0 {
		return pb.AppStatusCode_STATUS_CODE_ACTIVE
	}
	if stopped > 0 && active == 0 && idle == 0 {
		return pb.AppStatusCode_STATUS_CODE_STOPPED
	}
	if idle > 0 || (active > 0 && stopped > 0) {
		return pb.AppStatusCode_STATUS_CODE_IDLE
	}

	return pb.AppStatusCode_STATUS_CODE_UNKNOWN
}

// NewGetAppsStatusQueryHandler creates a new GetAppsStatusQueryHandler
func NewGetAppsStatusQueryHandler(ansible ansible.Repository) *GetAppsStatusQueryHandler {
	return &GetAppsStatusQueryHandler{
		ansible: ansible,
	}
}
