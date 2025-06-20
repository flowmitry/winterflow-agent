package repository

import (
	pkgansible "winterflow-agent/pkg/ansible"
)

// RunnerRepository defines the interface for executing deployment and management commands
type RunnerRepository interface {
	// DeployIngress runs the initial configuration playbook
	DeployIngress()

	// DeployApp deploys an application with the specified ID and version
	DeployApp(appID, appVersion string) pkgansible.Result

	// StopApp stops the application specified by the given app ID and returns the result of the operation.
	StopApp(appID string) pkgansible.Result

	// RestartApp restarts the specified application by its app ID and version and returns the result of the operation.
	RestartApp(appID, appVersion string) pkgansible.Result

	// UpdateApp updates the specified application by its app ID and version and returns the result of the operation.
	UpdateApp(appID string) pkgansible.Result

	// DeleteApp removes an application identified by the provided appID and returns the result of the operation.
	DeleteApp(appID string) pkgansible.Result

	// GenerateAppsStatus generate json files with the status of all applications
	GenerateAppsStatus(statusOutputPath string) pkgansible.Result
}
