package model

// GetAppStatusResult represents the result of a Docker operation
type GetAppStatusResult struct {
	App *ContainerApp
}

// GetAppsStatusResult represents the result of getting all apps status
type GetAppsStatusResult struct {
	Apps []*ContainerApp
}
