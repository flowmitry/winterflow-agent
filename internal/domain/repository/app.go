package repository

import (
	"winterflow-agent/internal/domain/model"
)

// AppRepository is an interface for managing Docker operations
type AppRepository interface {
	// GetAppStatus returns the status of a specific application
	GetAppStatus(appID string) (model.GetAppStatusResult, error)

	// GetAppsStatus returns the status of all available applications
	GetAppsStatus() (model.GetAppsStatusResult, error)

	// DeployApp deploys an application with the specified ID (deploys latest version)
	DeployApp(appID string) error

	// StartApp starts an application with the specified ID
	StartApp(appID string) error

	// StopApp stops the application specified by the given app ID.
	StopApp(appID string) error

	// RestartApp restarts the specified application by its app ID (latest version).
	RestartApp(appID string) error

	// UpdateApp updates the specified application by its app ID and version.
	UpdateApp(appID string) error

	// RenameApp renames an application identified by the given appID and returns an error if the operation fails.
	RenameApp(appID string, appName string) error

	// DeleteApp removes an application identified by the provided appID.
	DeleteApp(appID string) error

	// GetLogs retrieves logs for a specific application identified by appID.
	// The time range is defined by unix timestamps (seconds) in the `since` and `until` parameters.
	// A zero value disables the respective boundary (i.e. retrieve from the beginning or up to now).
	// The `tail` parameter limits the number of log lines returned. A value <= 0 returns all available logs.
	GetLogs(appID string, since int64, until int64, tail int32) (model.Logs, error)
}
