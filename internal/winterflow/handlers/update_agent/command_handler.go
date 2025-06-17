package update_agent

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"winterflow-agent/internal/config"
	agentversion "winterflow-agent/internal/version"
	log "winterflow-agent/pkg/log"
)

// UpdateAgentHandler handles the UpdateAgentCommand
type UpdateAgentHandler struct {
	config *config.Config
}

// Handle executes the UpdateAgentCommand
func (h *UpdateAgentHandler) Handle(cmd UpdateAgentCommand) error {
	if h.config.IsFeatureEnabled(config.FeatureAgentUpdateDisabled) {
		return log.Errorf("Update agent feature is disabled")
	}

	if cmd.Request == nil {
		return log.Errorf("invalid request: request is nil")
	}

	if cmd.Request.Base == nil {
		return log.Errorf("invalid request: base message is nil")
	}

	messageID := cmd.Request.Base.MessageId
	targetVersion := cmd.Request.Version
	log.Debug("Processing update agent request", "message_id", messageID, "current_version", agentversion.GetVersion(), "target_version", targetVersion)

	if targetVersion == "" {
		return log.Errorf("targetVersion is required for update agent command")
	}

	if agentversion.IsBiggerThan(targetVersion) {
		log.Info("Agent already uses newer version", "current_version", agentversion.GetVersion(), "target_version", targetVersion)
		return nil
	}

	// Get the current executable path
	execPath, err := os.Executable()
	if err != nil {
		return log.Errorf("failed to get current executable path: %w", err)
	}

	// Create a temporary directory for the download
	tempDir, err := os.MkdirTemp("", "winterflow-agent-update")
	if err != nil {
		return log.Errorf("failed to create temporary directory: %w", err)
	}
	defer os.RemoveAll(tempDir)

	// Construct the download URL
	// Format: {GitHubReleasesURL}/{targetVersion}/winterflow-agent-{os}-{arch}
	osName := runtime.GOOS
	archName := runtime.GOARCH
	binaryName := fmt.Sprintf("winterflow-agent-%s-%s", osName, archName)
	if osName == "windows" {
		return log.Errorf("windows is not supported")
	}

	downloadURL := fmt.Sprintf("%s/%s/%s", h.config.GetGitHubReleasesURL(), targetVersion, binaryName)
	log.Debug("Downloading agent version", "target_version", targetVersion, "url", downloadURL)

	// Download the binary
	resp, err := http.Get(downloadURL)
	if err != nil {
		return log.Errorf("failed to download agent binary: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return log.Errorf("failed to download agent binary, status code: %d", resp.StatusCode)
	}

	// Create a temporary file for the download
	tempFile := filepath.Join(tempDir, "winterflow-agent-new")

	out, err := os.Create(tempFile)
	if err != nil {
		return log.Errorf("failed to create temporary file: %w", err)
	}
	defer out.Close()

	// Set the file permissions to match the current executable
	info, err := os.Stat(execPath)
	if err != nil {
		return log.Errorf("failed to get current executable info: %w", err)
	}
	if err := os.Chmod(tempFile, info.Mode()); err != nil {
		return log.Errorf("failed to set file permissions: %w", err)
	}

	// Copy the downloaded binary to the temporary file
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return log.Errorf("failed to write downloaded binary: %w", err)
	}
	out.Close()

	log.Debug("Successfully downloaded agent version", "target_version", targetVersion, "file", tempFile)

	// On Unix-like systems, we can replace the executable and let systemd restart the service
	log.Debug("Replacing current executable with new version", "executable_path", execPath)
	if err := os.Rename(tempFile, execPath); err != nil {
		return log.Errorf("failed to replace current executable: %w", err)
	}

	log.Info("Successfully replaced agent with new version, exiting to let systemd restart the service", "current_version", agentversion.GetVersion(), "target_version", targetVersion)
	os.Exit(0)
	return nil
}

// NewUpdateAgentHandler creates a new UpdateAgentHandler
func NewUpdateAgentHandler(config *config.Config) *UpdateAgentHandler {
	return &UpdateAgentHandler{
		config: config,
	}
}
