package docker_compose

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"winterflow-agent/internal/application/config"
	"winterflow-agent/internal/domain/model"
	"winterflow-agent/internal/domain/repository"
	"winterflow-agent/internal/infra/docker"
	log "winterflow-agent/pkg/log"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
)

// composeRepository implements the Repository interface for Docker Compose
type composeRepository struct {
	client *client.Client
	mu     sync.RWMutex
	config *config.Config
}

// NewComposeRepository creates a new Docker Compose repository
func NewComposeRepository(config *config.Config, dockerClient *client.Client) repository.AppRepository {
	return &composeRepository{
		client: dockerClient,
		config: config,
	}
}

func (r *composeRepository) GetClient() *client.Client {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.client
}

func (r *composeRepository) GetAppStatus(ctx context.Context, appID string) (model.GetAppStatusResult, error) {
	appName, err := r.getAppName(appID)
	if err != nil {
		return model.GetAppStatusResult{}, fmt.Errorf("cannot get app status: %w", err)
	}
	log.Debug("Getting Docker Compose app status", "app_id", appID, "app_name", appName)

	// List containers with the app name as the project label filter
	filterArgs := filters.NewArgs()
	filterArgs.Add("label", fmt.Sprintf("com.docker.compose.project=%s", appName))

	options := container.ListOptions{
		All:     true,
		Filters: filterArgs,
	}

	dockerContainers, err := r.client.ContainerList(ctx, options)
	if err != nil {
		log.Error("Failed to list containers for app", "app_id", appID, "error", err)
		return model.GetAppStatusResult{}, fmt.Errorf("failed to list containers: %w", err)
	}

	// Create ContainerApp model
	containerApp := &model.ContainerApp{
		ID:         appID,
		Name:       appName,
		Containers: make([]model.Container, 0, len(dockerContainers)),
	}

	if len(dockerContainers) == 0 {
		log.Debug("No containers found for app", "app_id", appID)
		return model.GetAppStatusResult{App: containerApp}, nil
	}

	// Convert Docker containers to Container models
	for _, dockerContainer := range dockerContainers {
		containerInstance := model.Container{
			ID:         dockerContainer.ID,
			Name:       strings.TrimPrefix(dockerContainer.Names[0], "/"), // Remove leading slash
			StatusCode: docker.MapDockerStateToContainerStatus(dockerContainer.State),
			ExitCode:   0, // Docker API doesn't provide exit code in list response
			Ports:      docker.MapDockerPortsToContainerPorts(dockerContainer.Ports),
		}

		// Add error information for problematic containers
		if containerInstance.StatusCode == model.ContainerStatusProblematic {
			containerInstance.Error = fmt.Sprintf("Container in problematic state: %s", dockerContainer.Status)
		}

		containerApp.Containers = append(containerApp.Containers, containerInstance)
	}

	log.Debug("Docker Compose app status retrieved", "app_id", appID, "containers", len(containerApp.Containers))

	return model.GetAppStatusResult{App: containerApp}, nil
}

func (r *composeRepository) GetAppsStatus(ctx context.Context) (model.GetAppsStatusResult, error) {
	log.Debug("Getting Docker Compose apps status for all applications")

	// List all containers with Docker Compose project labels
	filterArgs := filters.NewArgs()
	filterArgs.Add("label", "com.docker.compose.project")

	options := container.ListOptions{
		All:     true,
		Filters: filterArgs,
	}

	dockerContainers, err := r.client.ContainerList(ctx, options)
	if err != nil {
		log.Error("Failed to list containers for all apps", "error", err)
		return model.GetAppsStatusResult{}, fmt.Errorf("failed to list containers: %w", err)
	}

	// Group containers by app ID (compose project)
	appContainers := make(map[string][]container.Summary)
	for _, dockerContainer := range dockerContainers {
		if appID, exists := dockerContainer.Labels["com.docker.compose.project"]; exists {
			appContainers[appID] = append(appContainers[appID], dockerContainer)
		}
	}

	// Create ContainerApp models for each app
	var apps []*model.ContainerApp
	for appID, containers := range appContainers {
		containerApp := &model.ContainerApp{
			ID:         appID,
			Name:       appID, // For Docker Compose, use project name as app name
			Containers: make([]model.Container, 0, len(containers)),
		}

		// Convert Docker containers to Container models
		for _, dockerContainer := range containers {
			containerInstance := model.Container{
				ID:         dockerContainer.ID,
				Name:       strings.TrimPrefix(dockerContainer.Names[0], "/"), // Remove leading slash
				StatusCode: docker.MapDockerStateToContainerStatus(dockerContainer.State),
				ExitCode:   0, // Docker API doesn't provide exit code in list response
				Ports:      docker.MapDockerPortsToContainerPorts(dockerContainer.Ports),
			}

			// Add error information for problematic containers
			if containerInstance.StatusCode == model.ContainerStatusProblematic {
				containerInstance.Error = fmt.Sprintf("Container in problematic state: %s", dockerContainer.Status)
			}

			containerApp.Containers = append(containerApp.Containers, containerInstance)
		}

		apps = append(apps, containerApp)
	}

	log.Debug("Docker Compose apps status retrieved", "apps_count", len(apps))

	return model.GetAppsStatusResult{Apps: apps}, nil
}

func (r *composeRepository) DeployApp(appID, appVersion string) error {
	// Determine important directories based on the agent configuration
	templateDir := filepath.Join(r.config.GetAppsTemplatesPath(), appID, appVersion)

	// Retrieve application name from the config to construct the output directory name
	appName, err := r.getAppName(appID)
	if err != nil {
		return fmt.Errorf("cannot deploy app: %w", err)
	}

	outputDir := filepath.Join(r.config.GetAppsPath(), appName)

	// Validate that the role directory exists
	if _, err := os.Stat(templateDir); err != nil {
		return fmt.Errorf("role directory %s does not exist: %w", templateDir, err)
	}

	// Ensure the output directory exists
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return fmt.Errorf("failed to create output directory %s: %w", outputDir, err)
	}

	// Load variables (defaults first, then override with vars)
	vars, err := r.loadTemplateVariables(templateDir)
	if err != nil {
		return fmt.Errorf("failed to load template variables: %w", err)
	}

	// Render templates into the output directory
	if err := r.renderTemplates(templateDir, outputDir, vars); err != nil {
		return fmt.Errorf("failed to render templates: %w", err)
	}

	// Finally start the application containers
	if err := r.composeUp(outputDir); err != nil {
		return fmt.Errorf("docker compose up failed: %w", err)
	}

	log.Info("[Deploy] successfully deployed app", "app_id", appID, "app_name", appName, "version", appVersion)
	return nil
}

func (r *composeRepository) StopApp(appID string) error {
	appName, err := r.getAppName(appID)
	if err != nil {
		return fmt.Errorf("cannot stop app: %w", err)
	}
	appDir := filepath.Join(r.config.GetAppsPath(), appName)
	if _, err := os.Stat(appDir); err != nil {
		if os.IsNotExist(err) {
			log.Warn("[Stop] app directory does not exist, skipping", "app_id", appID)
			return nil
		}
		return fmt.Errorf("failed to stat app directory: %w", err)
	}

	if err := r.composeDown(appDir); err != nil {
		return fmt.Errorf("docker compose down failed: %w", err)
	}

	log.Info("[Stop] successfully stopped app", "app_id", appID)
	return nil
}

func (r *composeRepository) RestartApp(appID, _ string) error {
	appName, err := r.getAppName(appID)
	if err != nil {
		return fmt.Errorf("cannot restart app: %w", err)
	}
	appDir := filepath.Join(r.config.GetAppsPath(), appName)
	if _, err := os.Stat(appDir); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("app directory %s does not exist", appDir)
		}
		return fmt.Errorf("failed to stat app directory: %w", err)
	}

	if err := r.composeRestart(appDir); err != nil {
		return fmt.Errorf("docker compose restart failed: %w", err)
	}

	log.Info("[Restart] successfully restarted app", "app_id", appID)
	return nil
}

func (r *composeRepository) UpdateApp(appID string) error {
	appName, err := r.getAppName(appID)
	if err != nil {
		return fmt.Errorf("cannot update app: %w", err)
	}
	appDir := filepath.Join(r.config.GetAppsPath(), appName)
	if _, err := os.Stat(appDir); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("app directory %s does not exist", appDir)
		}
		return fmt.Errorf("failed to stat app directory: %w", err)
	}

	// Pull the latest images first
	if err := r.composePull(appDir); err != nil {
		return fmt.Errorf("docker compose pull failed: %w", err)
	}

	// Recreate containers
	if err := r.composeUp(appDir); err != nil {
		return fmt.Errorf("docker compose up (after pull) failed: %w", err)
	}

	log.Info("[Update] successfully updated app", "app_id", appID)
	return nil
}

func (r *composeRepository) DeleteApp(appID string) error {
	// Stop the application first; ignore error if already stopped
	_ = r.StopApp(appID)
	log.Debug("[Delete] stopping app completed", "app_id", appID)

	// No other docker-specific resources need to be cleaned up here – files will be removed by the caller
	log.Info("[Delete] docker compose cleanup completed", "app_id", appID)
	return nil
}

// -----------------------------------------------------------------------------
// Helper functions
// -----------------------------------------------------------------------------

// loadTemplateVariables merges defaults and vars files into a single map.
func (r *composeRepository) loadTemplateVariables(templateDir string) (map[string]string, error) {
	vars := make(map[string]string)

	varsPath := filepath.Join(templateDir, "vars", "values.json")
	data, err := os.ReadFile(varsPath)
	if err != nil {
		if os.IsNotExist(err) {
			// No variables file – return empty map
			return vars, nil
		}
		return nil, err
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("failed to parse variables JSON: %w", err)
	}

	// Convert all values to string for simple template substitution
	for k, v := range raw {
		vars[k] = fmt.Sprintf("%v", v)
	}

	return vars, nil
}

// renderTemplates processes *.j2 files from roleDir/templates into destDir, performing a naive variable substitution.
func (r *composeRepository) renderTemplates(templateDir, destDir string, vars map[string]string) error {
	templatesPattern := filepath.Join(templateDir, "files", "*.j2")
	files, err := filepath.Glob(templatesPattern)
	if err != nil {
		return fmt.Errorf("failed to list template files: %w", err)
	}

	for _, src := range files {
		contentBytes, err := os.ReadFile(src)
		if err != nil {
			return fmt.Errorf("failed to read template %s: %w", src, err)
		}
		content := string(contentBytes)

		// Very naive substitution – replace {{ var }} and {{var}} occurrences
		for name, value := range vars {
			patterns := []string{
				fmt.Sprintf("{{ %s }}", name),
				fmt.Sprintf("{{%s}}", name),
			}
			for _, p := range patterns {
				content = strings.ReplaceAll(content, p, value)
			}
		}

		// Remove any leftover Jinja delimiters to avoid compose errors
		content = strings.ReplaceAll(content, "{{", "")
		content = strings.ReplaceAll(content, "}}", "")

		dstFilename := filepath.Base(src)
		dstFilename = strings.TrimSuffix(dstFilename, ".j2")
		dstPath := filepath.Join(destDir, dstFilename)
		if err := os.WriteFile(dstPath, []byte(content), 0o644); err != nil {
			return fmt.Errorf("failed to write rendered template to %s: %w", dstPath, err)
		}
	}
	return nil
}

// composeUp performs `docker compose up -d` using the most appropriate compose files present in appDir.
func (r *composeRepository) composeUp(appDir string) error {
	files, err := r.detectComposeFiles(appDir)
	if err != nil {
		return err
	}
	args := append(r.buildComposeFileArgs(files), "up", "-d")
	return r.runDockerCompose(appDir, args...)
}

func (r *composeRepository) composeDown(appDir string) error {
	files, err := r.detectComposeFiles(appDir)
	if err != nil {
		return err
	}
	args := append(r.buildComposeFileArgs(files), "down", "--remove-orphans")
	return r.runDockerCompose(appDir, args...)
}

func (r *composeRepository) composeRestart(appDir string) error {
	files, err := r.detectComposeFiles(appDir)
	if err != nil {
		return err
	}
	args := append(r.buildComposeFileArgs(files), "restart")
	return r.runDockerCompose(appDir, args...)
}

func (r *composeRepository) composePull(appDir string) error {
	files, err := r.detectComposeFiles(appDir)
	if err != nil {
		return err
	}
	args := append(r.buildComposeFileArgs(files), "pull")
	return r.runDockerCompose(appDir, args...)
}

// detectComposeFiles determines which compose files exist and returns them in the correct order
// mimicking the logic of the original Ansible playbooks.
func (r *composeRepository) detectComposeFiles(appDir string) ([]string, error) {
	var files []string

	custom := filepath.Join(appDir, "compose.custom.yml")
	override := filepath.Join(appDir, "compose.override.yml")
	compose := filepath.Join(appDir, "compose.yml")

	customExists := fileExists(custom)
	overrideExists := fileExists(override)
	composeExists := fileExists(compose)

	switch {
	case customExists && overrideExists:
		files = []string{custom, override}
	case customExists:
		files = []string{custom}
	case composeExists && overrideExists:
		files = []string{compose, override}
	case composeExists:
		// With standard compose.yml only we don't need to specify -f flag – docker compose will auto-detect it.
		// Return empty slice to indicate default behaviour.
		files = nil
	default:
		return nil, fmt.Errorf("no compose file found in %s", appDir)
	}
	return files, nil
}

// buildComposeFileArgs converts a slice of compose files into command-line arguments.
func (r *composeRepository) buildComposeFileArgs(files []string) []string {
	if len(files) == 0 {
		return nil
	}
	var args []string
	for _, f := range files {
		args = append(args, "-f", filepath.Base(f))
	}
	return args
}

// runDockerCompose executes `docker compose` with the provided arguments in the specified directory.
func (r *composeRepository) runDockerCompose(dir string, args ...string) error {
	fullCmd := append([]string{"compose"}, args...)
	cmd := exec.Command("docker", fullCmd...)
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Error("docker compose command failed", "dir", dir, "args", fullCmd, "output", string(output), "error", err)
		return fmt.Errorf("docker compose %v failed: %w", args, err)
	}
	log.Debug("docker compose executed", "dir", dir, "args", fullCmd, "output", string(output))
	return nil
}

// fileExists returns true if the given file exists and is not a directory.
func fileExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !info.IsDir()
}

// Helper getAppName reads app config and returns its Name field; falls back to appID if not found.
func (r *composeRepository) getAppName(appID string) (string, error) {
	// Build path to the current version config.json for the app
	configPath := filepath.Join(
		r.config.GetAppsTemplatesPath(),
		appID,
		r.config.GetAppsCurrentVersionFolder(),
		"config.json",
	)

	data, err := os.ReadFile(configPath)
	if err != nil {
		return appID, fmt.Errorf("failed to read app config: %w", err)
	}

	appConfig, err := model.ParseAppConfig(data)
	if err != nil {
		return appID, fmt.Errorf("failed to parse app config: %w", err)
	}

	if strings.TrimSpace(appConfig.Name) == "" {
		return "", fmt.Errorf("application name is empty in config for app %s", appID)
	}
	return appConfig.Name, nil
}

// Implement RenameApp method to satisfy AppRepository interface.
func (r *composeRepository) RenameApp(appID string, appName string) error {
	if appID == appName {
		// Nothing to do
		return nil
	}

	oldDir := filepath.Join(r.config.GetAppsPath(), appID)
	newDir := filepath.Join(r.config.GetAppsPath(), appName)

	// If old directory doesn't exist, nothing to rename
	if _, err := os.Stat(oldDir); os.IsNotExist(err) {
		return fmt.Errorf("app directory %s does not exist", oldDir)
	}

	// Ensure we don't overwrite an existing directory
	if _, err := os.Stat(newDir); err == nil {
		return fmt.Errorf("target app directory %s already exists", newDir)
	}

	// Step 1: stop running containers under the old project name
	if err := r.composeDown(oldDir); err != nil {
		return fmt.Errorf("failed to stop containers before rename: %w", err)
	}

	// Step 2: rename the directory
	if err := os.Rename(oldDir, newDir); err != nil {
		return fmt.Errorf("failed to rename app directory from %s to %s: %w", oldDir, newDir, err)
	}

	// Step 3: start containers under the new project name
	if err := r.composeUp(newDir); err != nil {
		return fmt.Errorf("failed to start containers after rename: %w", err)
	}

	log.Info("[Rename] successfully renamed and restarted app", "old_dir", oldDir, "new_dir", newDir)
	return nil
}
