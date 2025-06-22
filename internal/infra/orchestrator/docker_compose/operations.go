package docker_compose

import (
	"fmt"
	"os"
	"path/filepath"

	log "winterflow-agent/pkg/log"
)

// DeployApp renders templates for the given version of an application and starts the containers.
func (r *composeRepository) DeployApp(appID, appVersion string) error {
	templateDir := filepath.Join(r.config.GetAppsTemplatesPath(), appID, appVersion)

	appName, err := r.getAppName(appID)
	if err != nil {
		return fmt.Errorf("cannot deploy app: %w", err)
	}
	outputDir := filepath.Join(r.config.GetAppsPath(), appName)

	if _, err := os.Stat(templateDir); err != nil {
		return fmt.Errorf("role directory %s does not exist: %w", templateDir, err)
	}
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return fmt.Errorf("failed to create output directory %s: %w", outputDir, err)
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

	log.Info("[Deploy] successfully deployed app", "app_id", appID, "app_name", appName, "version", appVersion)
	return nil
}

// StopApp stops all containers belonging to the specified application.
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

// RestartApp restarts containers of the given application.
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

// UpdateApp pulls the latest images for the project and recreates containers.
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
	_ = r.StopApp(appID)
	log.Debug("[Delete] stopping app completed", "app_id", appID)
	log.Info("[Delete] docker compose cleanup completed", "app_id", appID)
	return nil
}

// RenameApp renames the compose project directory and restarts containers under the new name.
func (r *composeRepository) RenameApp(appID, appName string) error {
	if appID == appName {
		return nil
	}

	oldDir := filepath.Join(r.config.GetAppsPath(), appID)
	newDir := filepath.Join(r.config.GetAppsPath(), appName)

	if _, err := os.Stat(oldDir); os.IsNotExist(err) {
		return fmt.Errorf("app directory %s does not exist", oldDir)
	}
	if _, err := os.Stat(newDir); err == nil {
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

	log.Info("[Rename] successfully renamed and restarted app", "old_dir", oldDir, "new_dir", newDir)
	return nil
}
