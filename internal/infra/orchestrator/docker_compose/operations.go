package docker_compose

import (
	"fmt"
	"os"
	"path/filepath"

	"winterflow-agent/internal/domain/model"
	appsvc "winterflow-agent/internal/domain/service/app"
	"winterflow-agent/pkg/log"
)

// DeployApp renders templates for the given revision of an application and starts the containers.
func (r *composeRepository) DeployApp(appID string) error {
	// Ensure the base applications directory exists before proceeding.
	if err := ensureDir(r.config.GetAppsPath()); err != nil {
		return fmt.Errorf("failed to ensure apps base directory exists: %w", err)
	}

	versionService := appsvc.NewRevisionService(r.config)
	latest, err := versionService.GetLatestAppRevision(appID)
	if err != nil {
		return fmt.Errorf("failed to determine latest version for app %s: %w", appID, err)
	}

	templateDir := versionService.GetRevisionDir(appID, latest)
	appName, err := r.getAppName(templateDir)
	if err != nil {
		return fmt.Errorf("cannot deploy app: %w", err)
	}
	outputDir := filepath.Join(r.config.GetAppsPath(), appName)

	if _, err := os.Stat(templateDir); err != nil {
		return fmt.Errorf("role directory %s does not exist: %w", templateDir, err)
	}

	// If the application is already deployed, check if it's running and stop containers before we re-render.
	if dirExists(outputDir) {
		// Check if the service is running before attempting to stop it
		statusResult, statusErr := r.GetAppStatus(appID)
		containersAreRunning := false
		if statusErr == nil && statusResult.App != nil {
			code := statusResult.App.StatusCode
			containersAreRunning = code != model.ContainerStatusStopped && code != model.ContainerStatusUnknown
		} else if statusErr != nil {
			log.Warn("Unable to determine app status before deployment", "app_id", appID, "error", statusErr)
		}

		// Only stop containers if they are running
		if containersAreRunning {
			if err := r.composeDown(outputDir); err != nil {
				return fmt.Errorf("failed to stop running containers before deployment: %w", err)
			}
		}
	}

	// Render (or re-render) the application files on disk.
	if err := r.renderApp(appID, templateDir, outputDir); err != nil {
		return err
	}

	// Start containers using the freshly rendered project definition.
	if err := r.composeUp(outputDir); err != nil {
		return fmt.Errorf("docker compose up failed: %w", err)
	}

	log.Info("[Deploy] successfully deployed app", "app_id", appID, "app_name", appName, "version", latest)
	return nil
}

// StartApp starts an application with the specified ID (deploys latest version)
func (r *composeRepository) StartApp(appID string) error {
	// Ensure the base applications directory exists before proceeding.
	if err := ensureDir(r.config.GetAppsPath()); err != nil {
		return fmt.Errorf("failed to ensure apps base directory exists: %w", err)
	}

	// Resolve the human-readable application name from the latest version's config.
	appName, err := r.getAppNameById(appID)
	if err != nil {
		// Could not resolve name – fall back to a full deploy which will surface a clearer error.
		return r.DeployApp(appID)
	}
	outputDir := filepath.Join(r.config.GetAppsPath(), appName)

	// If the app hasn't been rendered yet, perform a full deploy (render + start).
	if !dirExists(outputDir) {
		return r.DeployApp(appID)
	}

	// Start (or resume) the containers for the already rendered project.
	if err := r.composeUp(outputDir); err != nil {
		return fmt.Errorf("docker compose up failed: %w", err)
	}

	log.Info("[Start] successfully started app", "app_id", appID)
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

	appName, err := r.getAppNameById(appID)
	if err != nil {
		// If we cannot resolve the name it likely means the app hasn't been rendered yet – deploy it.
		return r.DeployApp(appID)
	}
	appDir := filepath.Join(r.config.GetAppsPath(), appName)

	// If the application directory does not exist, fall back to a full deploy (render + start).
	if !dirExists(appDir) {
		return r.DeployApp(appID)
	}

	// Perform an in-place container restart.
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

	statusResult, statusErr := r.GetAppStatus(appID)
	containersWereRunning := false
	if statusErr == nil && statusResult.App != nil {
		code := statusResult.App.StatusCode
		containersWereRunning = code != model.ContainerStatusStopped && code != model.ContainerStatusUnknown
	} else if statusErr != nil {
		log.Warn("Unable to determine app status before rename", "app_id", appID, "error", statusErr)
	}

	// Always attempt to cleanly stop containers (if any) before the directory move
	if err := r.composeDown(oldDir); err != nil {
		return fmt.Errorf("failed to stop containers before rename: %w", err)
	}

	if err := os.Rename(oldDir, newDir); err != nil {
		return fmt.Errorf("failed to rename app directory from %s to %s: %w", oldDir, newDir, err)
	}

	// Restart containers only when they were running prior to the rename.
	if containersWereRunning {
		if err := r.composeUp(newDir); err != nil {
			return fmt.Errorf("failed to start containers after rename: %w", err)
		}
	}

	if containersWereRunning {
		log.Info("[Rename] successfully renamed and restarted app", "app_id", appID, "old_name", oldName, "new_name", appName)
	} else {
		log.Info("[Rename] successfully renamed app (was stopped)", "app_id", appID, "old_name", oldName, "new_name", appName)
	}
	return nil
}
