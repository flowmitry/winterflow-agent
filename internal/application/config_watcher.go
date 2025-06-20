package application

import (
	"context"
	"time"
	"winterflow-agent/internal/config"
	"winterflow-agent/pkg/files"
	log "winterflow-agent/pkg/log"
)

// ConfigWatcher watches a configuration file for changes
type ConfigWatcher struct {
	fileWatcher *files.FileWatcher
	onChange    func(*config.Config)
}

// NewConfigWatcher creates a new configuration file watcher
func NewConfigWatcher(configPath string, onChange func(*config.Config)) *ConfigWatcher {
	cw := &ConfigWatcher{
		onChange: onChange,
	}

	// Create the file watcher with our config-specific callback
	cw.fileWatcher = files.NewFileWatcher(configPath, cw.handleFileChange)

	return cw
}

// Start begins watching the configuration file for changes
func (w *ConfigWatcher) Start(ctx context.Context) error {
	log.Info("Config watcher starting for %s", w.fileWatcher.GetFilePath())
	return w.fileWatcher.Start(ctx)
}

// Stop stops watching the configuration file
func (w *ConfigWatcher) Stop() {
	log.Info("Config watcher stopping")
	w.fileWatcher.Stop()
}

// handleFileChange handles file change events and loads the new configuration
func (w *ConfigWatcher) handleFileChange(filePath string) {
	log.Info("Configuration file changed, reloading")

	// Load the new configuration
	newConfig, err := config.LoadConfig(filePath)
	if err != nil {
		log.Error("Failed to load new configuration: %v", err)
		return
	}

	// Call the onChange callback with the new configuration
	if w.onChange != nil {
		w.onChange(newConfig)
	}
}

// SetInterval sets the interval for checking file changes
func (w *ConfigWatcher) SetInterval(interval time.Duration) {
	w.fileWatcher.SetInterval(interval)
}
