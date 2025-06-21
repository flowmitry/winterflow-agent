package repository

import (
	"context"
	"winterflow-agent/internal/domain/model"
)

// AppRepository is an interface for managing Docker operations
type AppRepository interface {
	// GetAppStatus returns the status of a specific application
	GetAppStatus(ctx context.Context, appID string) (model.GetAppStatusResult, error)

	// GetAppsStatus returns the status of all available applications
	GetAppsStatus(ctx context.Context) (model.GetAppsStatusResult, error)

	// DeployApp deploys an application with the specified ID and version
	DeployApp(appID, appVersion string) error

	// StopApp stops the application specified by the given app ID.
	StopApp(appID string) error

	// RestartApp restarts the specified application by its app ID and version.
	RestartApp(appID, appVersion string) error

	// UpdateApp updates the specified application by its app ID and version.
	UpdateApp(appID string) error

	// DeleteApp removes an application identified by the provided appID.
	DeleteApp(appID string) error
}
