package orchestrator

import (
    "fmt"
    "os"
    "path/filepath"
    "strings"
    "winterflow-agent/internal/application/config"
    "winterflow-agent/internal/domain/model"

    "github.com/docker/docker/api/types/container"
)

// MapDockerStateToContainerStatus maps Docker container state to ContainerStatusCode
func MapDockerStateToContainerStatus(state string) model.ContainerStatusCode {
    switch strings.ToLower(state) {
    case "running":
        return model.ContainerStatusActive
    case "exited", "stopped":
        return model.ContainerStatusStopped
    case "restarting":
        return model.ContainerStatusRestarting
    case "paused":
        return model.ContainerStatusIdle
    case "dead", "oomkilled":
        return model.ContainerStatusProblematic
    default:
        return model.ContainerStatusUnknown
    }
}

// MapDockerPortsToContainerPorts converts Docker ports to ContainerPort slice
func MapDockerPortsToContainerPorts(dockerPorts []container.Port) []model.ContainerPort {
    var ports []model.ContainerPort
    for _, dockerPort := range dockerPorts {
        if dockerPort.PublicPort > 0 {
            ports = append(ports, model.ContainerPort{
                Port:     int(dockerPort.PublicPort),
                Protocol: dockerPort.Type,
            })
        }
    }
    return ports
}

// SaveCurrentConfigCopy creates/updates a lightweight copy of the configuration that is currently
// being deployed. It copies <templateDir>/config.json into
//
//	apps_templates/{app_id}/current.config.json
//
// so that other system components can quickly inspect the active configuration without having to
// resolve versions.
//
// The function is orchestration-agnostic – it operates purely on the file system and therefore sits
// at the generic orchestrator layer rather than inside a concrete implementation such as
// docker_compose.
func SaveCurrentConfigCopy(cfg *config.Config, appID, templateDir string) error {
    srcConfigPath := filepath.Join(templateDir, "config.json")

    data, err := os.ReadFile(srcConfigPath)
    if err != nil {
        return fmt.Errorf("failed to read source configuration %s: %w", srcConfigPath, err)
    }

    dstConfigPath := filepath.Join(cfg.GetAppsTemplatesPath(), appID, "current.config.json")

    // Ensure destination directory exists.
    if err := os.MkdirAll(filepath.Dir(dstConfigPath), 0o755); err != nil {
        return fmt.Errorf("failed to create directory for current configuration: %w", err)
    }

    if err := os.WriteFile(dstConfigPath, data, 0o644); err != nil {
        return fmt.Errorf("failed to write current configuration copy: %w", err)
    }

    return nil
}

// GetCurrentConfig loads and parses the current.config.json for an application.
// It returns a parsed *model.AppConfig representing the configuration that was
// last deployed (i.e. written by SaveCurrentConfigCopy).
//
// The helper centralises path resolution and JSON parsing so that callers do
// not need to duplicate this logic across the codebase.
func GetCurrentConfig(appsTemplatesPath, appID string) (*model.AppConfig, error) {
    currentCfgPath := filepath.Join(appsTemplatesPath, appID, "current.config.json")

    data, err := os.ReadFile(currentCfgPath)
    if err != nil {
        return nil, fmt.Errorf("failed to read current config %s: %w", currentCfgPath, err)
    }

    appConfig, err := model.ParseAppConfig(data)
    if err != nil {
        return nil, fmt.Errorf("failed to parse current config: %w", err)
    }

    return appConfig, nil
}
