package docker_compose

import (
    "fmt"
    "os"
    "path/filepath"
    appsvc "winterflow-agent/internal/domain/service/app"
    log "winterflow-agent/pkg/log"
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

    // Clean up any previously rendered templates to avoid leaving obsolete files.
    if err := os.RemoveAll(outputDir); err != nil && !os.IsNotExist(err) {
        return fmt.Errorf("failed to clean output directory %s: %w", outputDir, err)
    }

    // Recreate the (now empty) output directory.
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
    if err := ensureDir(r.config.GetAppsPath()); err != nil {
        return fmt.Errorf("failed to ensure apps base directory exists: %w", err)
    }

    versionService := appsvc.NewAppVersionService(r.config)
    latest, err := versionService.GetLatestAppVersion(appID)
    if err != nil {
        return fmt.Errorf("failed to determine latest version for app %s: %w", appID, err)
    }
    templateDir := versionService.GetVersionDir(appID, latest)

    if _, err := os.Stat(templateDir); err != nil {
        return fmt.Errorf("template directory %s does not exist: %w", templateDir, err)
    }

    appName, err := r.getAppName(templateDir)
    if err != nil {
        return fmt.Errorf("cannot restart app: %w", err)
    }
    appDir := filepath.Join(r.config.GetAppsPath(), appName)

    // Stop running containers (ignore missing directory case since we may be deploying for the first time).
    if dirExists(appDir) {
        if err := r.composeDown(appDir); err != nil {
            return fmt.Errorf("failed to stop containers before restart: %w", err)
        }
    }

    // Ensure fresh files.
    if err := os.RemoveAll(appDir); err != nil && !os.IsNotExist(err) {
        return fmt.Errorf("failed to clean output directory %s: %w", appDir, err)
    }
    if err := os.MkdirAll(appDir, 0o755); err != nil {
        return fmt.Errorf("failed to recreate output directory %s: %w", appDir, err)
    }

    // Load variables and render templates recursively.
    vars, err := r.loadTemplateVariables(templateDir)
    if err != nil {
        return fmt.Errorf("failed to load template variables: %w", err)
    }
    if err := r.renderTemplates(templateDir, appDir, vars); err != nil {
        return fmt.Errorf("failed to render templates: %w", err)
    }

    // Start containers with updated files.
    if err := r.composeUp(appDir); err != nil {
        return fmt.Errorf("docker compose up failed after restart: %w", err)
    }

    log.Info("[Restart] successfully restarted app with updated files", "app_id", appID, "version", latest)
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

    log.Info("[Rename] successfully renamed and restarted app", "app_id", appID, "old_name", oldName, "new_name", appName)
    return nil
}
