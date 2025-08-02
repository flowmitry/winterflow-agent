package docker_compose

import (
	"fmt"
	"os"
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
	outputDir := r.getAppDir(appID)

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

	log.Info("[Deploy] successfully deployed app", "app_id", appID, "version", latest)
	return nil
}

// StartApp starts an application with the specified ID (deploys latest version)
func (r *composeRepository) StartApp(appID string) error {
	// Ensure the base applications directory exists before proceeding.
	if err := ensureDir(r.config.GetAppsPath()); err != nil {
		return fmt.Errorf("failed to ensure apps base directory exists: %w", err)
	}

	// Get the app directory path using the app ID directly
	outputDir := r.getAppDir(appID)

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
	appDir := r.getAppDir(appID)

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

	appDir := r.getAppDir(appID)

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
	appDir := r.getAppDir(appID)
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

// DeleteApp stops containers and removes the application directory.
func (r *composeRepository) DeleteApp(appID string) error {
	// Ensure the base applications directory exists.
	if err := ensureDir(r.config.GetAppsPath()); err != nil {
		return fmt.Errorf("failed to ensure apps base directory exists: %w", err)
	}

	// Get the app directory path using the app ID directly
	appDir := r.getAppDir(appID)

	// Check if the app directory exists
	if !dirExists(appDir) {
		log.Warn("[Delete] app directory does not exist, skipping", "app_id", appID, "app_dir", appDir)
		return nil
	}

	// Check if containers are running before attempting to stop them
	statusResult, statusErr := r.GetAppStatus(appID)
	containersAreRunning := false
	if statusErr == nil && statusResult.App != nil {
		code := statusResult.App.StatusCode
		containersAreRunning = code != model.ContainerStatusStopped && code != model.ContainerStatusUnknown
	} else if statusErr != nil {
		log.Warn("Unable to determine app status before deletion", "app_id", appID, "error", statusErr)
	}

	// Only attempt to stop containers if they are running
	if containersAreRunning {
		if err := r.StopApp(appID); err != nil {
			log.Warn("Failed to stop app before deletion, continuing with removal", "app_id", appID, "error", err)
		}
	}

	// Remove the app directory
	if err := os.RemoveAll(appDir); err != nil {
		return fmt.Errorf("failed to delete app directory for app ID %s: %w", appID, err)
	}

	log.Info("[Delete] successfully deleted app", "app_id", appID)
	return nil
}

func (r *composeRepository) RenameApp(appID, newName string) error {
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

	err = r.changeTemplateAppName(newName, templateDir)
	if err != nil {
		return fmt.Errorf("Failed to update a template revision: %w", err)
	}

	outputDir := r.getAppDir(appID)

	if _, err := os.Stat(templateDir); err != nil {
		return fmt.Errorf("role directory %s does not exist: %w", templateDir, err)
	}

	wasRunning := false

	// If the application is already deployed, check if it's running and stop containers before we re-render.
	if dirExists(outputDir) {
		// Check if the service is running before attempting to stop it
		statusResult, statusErr := r.GetAppStatus(appID)
		containersAreRunning := false
		if statusErr == nil && statusResult.App != nil {
			code := statusResult.App.StatusCode
			containersAreRunning = code != model.ContainerStatusStopped && code != model.ContainerStatusUnknown
			wasRunning = containersAreRunning
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

	if wasRunning {
		if err := r.composeUp(outputDir); err != nil {
			return fmt.Errorf("docker compose up failed: %w", err)
		}
	}

	log.Info("[Deploy] successfully renamed app", "app_id", appID, "version", latest, "template_dir", templateDir, "output_dir", outputDir, "wasRunning", wasRunning)
	return nil
}
