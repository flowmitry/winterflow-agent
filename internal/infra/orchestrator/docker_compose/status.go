package docker_compose

import (
    "context"
    "fmt"
    "os"
    "path/filepath"
    "strings"

    "winterflow-agent/internal/domain/model"
    "winterflow-agent/internal/infra/orchestrator"
    "winterflow-agent/pkg/log"

    "github.com/docker/docker/api/types/container"
    "github.com/docker/docker/api/types/filters"
)

// GetAppStatus returns detailed information for a single application identified by appID.
func (r *composeRepository) GetAppStatus(ctx context.Context, appID string) (model.GetAppStatusResult, error) {
    appName, err := r.getAppNameById(appID)
    if err != nil {
        return model.GetAppStatusResult{}, fmt.Errorf("cannot get app status: %w", err)
    }
    log.Debug("Getting Docker Compose app status", "app_id", appID, "app_name", appName)

    // Check if the application directory exists. This helps us distinguish between
    // a stopped application (directory exists but no containers) and an unknown one.
    appDir := filepath.Join(r.config.GetAppsPath(), appName)
    appDirExists := dirExists(appDir)

    // List containers that belong to the compose project.
    filterArgs := filters.NewArgs()
    filterArgs.Add("label", fmt.Sprintf("com.docker.compose.project=%s", appName))

    dockerContainers, err := r.client.ContainerList(ctx, container.ListOptions{All: true, Filters: filterArgs})
    if err != nil {
        log.Error("Failed to list containers for app", "app_id", appID, "error", err)
        return model.GetAppStatusResult{}, fmt.Errorf("failed to list containers: %w", err)
    }

    containerApp := &model.ContainerApp{
        ID:         appID,
        Name:       appName,
        Containers: make([]model.Container, 0, len(dockerContainers)),
    }

    for _, dockerContainer := range dockerContainers {
        c := model.Container{
            ID:         dockerContainer.ID,
            Name:       strings.TrimPrefix(dockerContainer.Names[0], "/"),
            StatusCode: orchestrator.MapDockerStateToContainerStatus(dockerContainer.State),
        }
        if c.StatusCode == model.ContainerStatusProblematic {
            c.Error = fmt.Sprintf("Container in problematic state: %s", dockerContainer.Status)
        }
        containerApp.Containers = append(containerApp.Containers, c)
    }

    // Derive overall status.
    if len(containerApp.Containers) == 0 {
        if appDirExists {
            containerApp.StatusCode = model.ContainerStatusStopped
        } else {
            containerApp.StatusCode = model.ContainerStatusUnknown
        }
    } else {
        containerApp.StatusCode = determineContainerAppStatus(containerApp.Containers)
    }

    log.Debug("Docker Compose app status retrieved", "app_id", appID, "containers", len(containerApp.Containers), "status_code", containerApp.StatusCode)
    return model.GetAppStatusResult{App: containerApp}, nil
}

// GetAppsStatus enumerates all compose projects on the host and returns aggregated status information.
func (r *composeRepository) GetAppsStatus(ctx context.Context) (model.GetAppsStatusResult, error) {
    log.Debug("Getting Docker Compose apps status for available applications")

    // 1. Determine list of available application IDs from templates directory.
    templatesDir := r.config.GetAppsTemplatesPath()
    entries, err := os.ReadDir(templatesDir)
    if err != nil {
        log.Error("Failed to read apps templates directory", "path", templatesDir, "error", err)
        return model.GetAppsStatusResult{}, fmt.Errorf("failed to read apps templates directory: %w", err)
    }

    var apps []*model.ContainerApp

    for _, entry := range entries {
        if !entry.IsDir() {
            continue // skip files
        }

        appID := entry.Name()

        // Re-use the detailed GetAppStatus for each available app. This ensures
        // consistent status calculation logic and avoids code duplication.
        statusResult, err := r.GetAppStatus(ctx, appID)
        if err != nil {
            // Log the error but continue processing the remaining apps – one
            // misconfigured app should not prevent other statuses from being returned.
            log.Warn("Failed to get status for app", "app_id", appID, "error", err)
            continue
        }

        if statusResult.App != nil {
            apps = append(apps, statusResult.App)
        }
    }

    log.Debug("Docker Compose apps status retrieved", "apps_count", len(apps))
    return model.GetAppsStatusResult{Apps: apps}, nil
}

// determineContainerAppStatus analyses containers and calculates an overall
// status for the application.
func determineContainerAppStatus(containers []model.Container) model.ContainerStatusCode {
    if len(containers) == 0 {
        return model.ContainerStatusStopped
    }

    var active, idle, stopped, restarting, problematic int
    for _, c := range containers {
        switch c.StatusCode {
        case model.ContainerStatusActive:
            active++
        case model.ContainerStatusIdle:
            idle++
        case model.ContainerStatusStopped:
            stopped++
        case model.ContainerStatusRestarting:
            if c.ExitCode != 0 {
                problematic++
            } else {
                restarting++
            }
        case model.ContainerStatusProblematic:
            problematic++
        default:
            problematic++
        }
    }

    if problematic > 0 {
        return model.ContainerStatusProblematic
    }
    if restarting > 0 {
        return model.ContainerStatusRestarting
    }
    if active > 0 && stopped == 0 && idle == 0 {
        return model.ContainerStatusActive
    }
    if stopped > 0 && active == 0 && idle == 0 {
        return model.ContainerStatusStopped
    }
    if idle > 0 || (active > 0 && stopped > 0) {
        return model.ContainerStatusIdle
    }
    return model.ContainerStatusUnknown
}
