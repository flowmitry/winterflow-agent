package docker_compose

import (
	"fmt"
	"os"
	"path/filepath"

	"winterflow-agent/internal/domain/model"
	appsvc "winterflow-agent/internal/domain/service/app"
	"winterflow-agent/internal/infra/orchestrator"
	"winterflow-agent/pkg/log"
)

// DeployApp renders templates for the given version of an application and starts the containers.
func (r *composeRepository) DeployApp(appID string) error {
	// Ensure the base applications directory exists before proceeding.
	if err := ensureDir(r.config.GetAppsPath()); err != nil {
		return fmt.Errorf("failed to ensure apps base directory exists: %w", err)
	}

	versionService := appsvc.NewAppVersionService(r.config)
	latest, err := versionService.GetLatestAppVersion(appID)
	if err != nil {
		return fmt.Errorf("failed to determine latest version for app %s: %w", appID, err)
	}

	templateDir := versionService.GetVersionDir(appID, latest)
	appName, err := r.getAppName(templateDir)
	if err != nil {
		return fmt.Errorf("cannot deploy app: %w", err)
	}
	outputDir := filepath.Join(r.config.GetAppsPath(), appName)

	if _, err := os.Stat(templateDir); err != nil {
		return fmt.Errorf("role directory %s does not exist: %w", templateDir, err)
	}

	// If the application is already deployed, stop running containers before deleting files.
	if dirExists(outputDir) {
		if err := r.composeDown(outputDir); err != nil {
			return fmt.Errorf("failed to stop running containers before deployment: %w", err)
		}
	}

	// Load the configuration of the version that is about to be deployed so we can compare it with
	// the configuration of the currently running version (if any).
	var newCfg *model.AppConfig
	{
		cfgPath := filepath.Join(templateDir, "config.json")
		data, err := os.ReadFile(cfgPath)
		if err != nil {
			return fmt.Errorf("failed to read new configuration %s: %w", cfgPath, err)
		}
		parsed, err := model.ParseAppConfig(data)
		if err != nil {
			return fmt.Errorf("failed to parse new configuration: %w", err)
		}
		newCfg = parsed
	}

	// Remove only the files that belong to the *currently* deployed version but are **absent** in
	// the new version instead of wiping the whole directory. This helps preserve any other data
	// that might have been generated or mounted in the application directory at runtime.
	if currentCfg, errCfg := orchestrator.GetCurrentConfig(r.config.GetAppsTemplatesPath(), appID); errCfg == nil {
		if err := r.removeDeployedFiles(outputDir, currentCfg, newCfg); err != nil {
			return fmt.Errorf("failed to remove previously deployed files: %w", err)
		}
	} else if !os.IsNotExist(errCfg) {
		// An error other than "file does not exist" indicates an unexpected problem – surface it.
		log.Warn("failed to load current configuration", "error", errCfg)
	}

	// Ensure the output directory exists in case it was not present before.
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return fmt.Errorf("failed to ensure output directory %s: %w", outputDir, err)
	}

	vars, err := r.loadTemplateVariables(templateDir)
	if err != nil {
		return fmt.Errorf("failed to load template variables: %w", err)
	}

	if err := r.renderTemplates(templateDir, outputDir, vars); err != nil {
		return fmt.Errorf("failed to render templates: %w", err)
	}

	if err := r.composeUp(outputDir); err != nil {
		return fmt.Errorf("docker compose up failed: %w", err)
	}

	// Save a copy of the configuration being deployed so that external tools can
	// easily inspect the active configuration without resolving version
	// directories.
	if err := orchestrator.SaveCurrentConfigCopy(r.config, appID, templateDir); err != nil {
		return err
	}

	log.Info("[Deploy] successfully deployed app", "app_id", appID, "app_name", appName, "version", latest)
	return nil
}

// StopApp stops all containers belonging to the specified application.
func (r *composeRepository) StopApp(appID string) error {
	// Ensure the base applications directory exists.
	if err := ensureDir(r.config.GetAppsPath()); err != nil {
		return fmt.Errorf("failed to ensure apps base directory exists: %w", err)
	}
	appName, err := r.getAppNameById(appID)
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

// RestartApp restarts containers of the given application.
func (r *composeRepository) RestartApp(appID string) error {
	// Ensure the base applications directory exists.
	if err := ensureDir(r.config.GetAppsPath()); err != nil {
		return fmt.Errorf("failed to ensure apps base directory exists: %w", err)
	}

	// Resolve the human-readable application name from its ID.
	appName, err := r.getAppNameById(appID)
	if err != nil {
		return fmt.Errorf("cannot restart app: %w", err)
	}
	appDir := filepath.Join(r.config.GetAppsPath(), appName)

	// Verify that the compose project directory exists before attempting the restart.
	if _, err := os.Stat(appDir); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("app directory %s does not exist", appDir)
		}
		return fmt.Errorf("failed to stat app directory: %w", err)
	}

	// Execute `docker compose restart` for the application.
	if err := r.composeRestart(appDir); err != nil {
		return fmt.Errorf("docker compose restart failed: %w", err)
	}

	log.Info("[Restart] successfully restarted app", "app_id", appID)
	return nil
}

// UpdateApp pulls the latest images for the project and recreates containers.
func (r *composeRepository) UpdateApp(appID string) error {
	if err := ensureDir(r.config.GetAppsPath()); err != nil {
		return fmt.Errorf("failed to ensure apps base directory exists: %w", err)
	}
	appName, err := r.getAppNameById(appID)
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

	if err := r.composePull(appDir); err != nil {
		return fmt.Errorf("docker compose pull failed: %w", err)
	}
	if err := r.composeUp(appDir); err != nil {
		return fmt.Errorf("docker compose up (after pull) failed: %w", err)
	}

	log.Info("[Update] successfully updated app", "app_id", appID)
	return nil
}

// DeleteApp stops containers (ignoring errors) – additional cleanup is handled elsewhere.
func (r *composeRepository) DeleteApp(appID string) error {
	if err := ensureDir(r.config.GetAppsPath()); err != nil {
		return fmt.Errorf("failed to ensure apps base directory exists: %w", err)
	}
	_ = r.StopApp(appID)
	log.Debug("[Delete] stopping app completed", "app_id", appID)
	log.Info("[Delete] docker compose cleanup completed", "app_id", appID)
	return nil
}

// RenameApp renames the compose project directory and restarts containers under the new name.
func (r *composeRepository) RenameApp(appID, appName string) error {
	if err := ensureDir(r.config.GetAppsPath()); err != nil {
		return fmt.Errorf("failed to ensure apps base directory exists: %w", err)
	}

	oldName, err := r.getAppNameById(appID)
	if err != nil {
		return fmt.Errorf("cannot rename app: %w", err)
	}

	if oldName == appName {
		return nil
	}

	oldDir := filepath.Join(r.config.GetAppsPath(), oldName)
	if !dirExists(oldDir) {
		return nil
	}

	newDir := filepath.Join(r.config.GetAppsPath(), appName)
	if dirExists(newDir) {
		return fmt.Errorf("target app directory %s already exists", newDir)
	}

	if err := r.composeDown(oldDir); err != nil {
		return fmt.Errorf("failed to stop containers before rename: %w", err)
	}

	if err := os.Rename(oldDir, newDir); err != nil {
		return fmt.Errorf("failed to rename app directory from %s to %s: %w", oldDir, newDir, err)
	}

	if err := r.composeUp(newDir); err != nil {
		return fmt.Errorf("failed to start containers after rename: %w", err)
	}

	log.Info("[Rename] successfully renamed and restarted app", "app_id", appID, "old_name", oldName, "new_name", appName)
	return nil
}
