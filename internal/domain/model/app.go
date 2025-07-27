package model

// App represents an application with its configuration, variables, and files
type App struct {
	ID        string
	Config    *AppConfig
	Variables VariableMap
	Files     FilesMap
}

// VariableMap represents a map of variable UUIDs to values
type VariableMap map[string]string

// FilesMap represents a map of variable UUIDs to values
type FilesMap map[string][]byte

// AppDetails represents the details of an application.
// It contains the application, the revision and the list of available revisions.
type AppDetails struct {
	App       *App
	Revision  uint32
	Revisions []uint32
}
