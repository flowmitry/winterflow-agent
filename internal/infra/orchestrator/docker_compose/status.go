package docker_compose

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"winterflow-agent/internal/domain/model"
	"winterflow-agent/internal/infra/orchestrator"
	log "winterflow-agent/pkg/log"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
)

// GetAppStatus returns detailed information for a single application identified by appID.
func (r *composeRepository) GetAppStatus(ctx context.Context, appID string) (model.GetAppStatusResult, error) {
	appName, err := r.getAppName(appID)
	if err != nil {
		return model.GetAppStatusResult{}, fmt.Errorf("cannot get app status: %w", err)
	}
	log.Debug("Getting Docker Compose app status", "app_id", appID, "app_name", appName)

	// Check if the application directory exists. This helps us distinguish between
	// a stopped application (directory exists but no containers) and an unknown one.
	appDir := filepath.Join(r.config.GetAppsPath(), appName)
	dirExists := fileExists(appDir)

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
			ExitCode:   0, // Not available in list response
			Ports:      orchestrator.MapDockerPortsToContainerPorts(dockerContainer.Ports),
		}
		if c.StatusCode == model.ContainerStatusProblematic {
			c.Error = fmt.Sprintf("Container in problematic state: %s", dockerContainer.Status)
		}
		containerApp.Containers = append(containerApp.Containers, c)
	}

	// Derive overall status.
	if len(containerApp.Containers) == 0 {
		if dirExists {
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
	log.Debug("Getting Docker Compose apps status for all applications")

	filterArgs := filters.NewArgs()
	filterArgs.Add("label", "com.docker.compose.project")

	dockerContainers, err := r.client.ContainerList(ctx, container.ListOptions{All: true, Filters: filterArgs})
	if err != nil {
		log.Error("Failed to list containers for all apps", "error", err)
		return model.GetAppsStatusResult{}, fmt.Errorf("failed to list containers: %w", err)
	}

	// Group containers by project label.
	grouped := make(map[string][]container.Summary)
	for _, dc := range dockerContainers {
		if project, ok := dc.Labels["com.docker.compose.project"]; ok {
			grouped[project] = append(grouped[project], dc)
		}
	}

	var apps []*model.ContainerApp
	for project, containers := range grouped {
		app := &model.ContainerApp{
			ID:         project,
			Name:       project,
			Containers: make([]model.Container, 0, len(containers)),
		}

		for _, dc := range containers {
			c := model.Container{
				ID:         dc.ID,
				Name:       strings.TrimPrefix(dc.Names[0], "/"),
				StatusCode: orchestrator.MapDockerStateToContainerStatus(dc.State),
				ExitCode:   0,
				Ports:      orchestrator.MapDockerPortsToContainerPorts(dc.Ports),
			}
			if c.StatusCode == model.ContainerStatusProblematic {
				c.Error = fmt.Sprintf("Container in problematic state: %s", dc.Status)
			}
			app.Containers = append(app.Containers, c)
		}

		// Overall status for this project.
		if len(app.Containers) == 0 {
			appDir := filepath.Join(r.config.GetAppsPath(), project)
			if fileExists(appDir) {
				app.StatusCode = model.ContainerStatusStopped
			} else {
				app.StatusCode = model.ContainerStatusUnknown
			}
		} else {
			app.StatusCode = determineContainerAppStatus(app.Containers)
		}

		apps = append(apps, app)
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
