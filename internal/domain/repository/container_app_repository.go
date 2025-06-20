package repository

import (
	"context"
	"winterflow-agent/internal/domain/model"
)

// ContainerAppRepository is an interface for managing Docker operations
type ContainerAppRepository interface {
	// GetAppStatus returns the status of a specific application
	GetAppStatus(ctx context.Context, appID string) (model.GetAppStatusResult, error)

	// GetAppsStatus returns the status of all available applications
	GetAppsStatus(ctx context.Context) (model.GetAppsStatusResult, error)
}
