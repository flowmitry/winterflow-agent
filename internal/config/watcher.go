package config

import (
	"context"
	"os"
	"sync"
	"time"
	log "winterflow-agent/pkg/log"
)

// ConfigWatcher watches a configuration file for changes
type ConfigWatcher struct {
	configPath string
	lastMod    time.Time
	interval   time.Duration
	onChange   func(*Config)
	stopCh     chan struct{}
	mu         sync.Mutex
	wg         sync.WaitGroup
}

// NewConfigWatcher creates a new configuration file watcher
func NewConfigWatcher(configPath string, onChange func(*Config)) *ConfigWatcher {
	return &ConfigWatcher{
		configPath: configPath,
		interval:   5 * time.Second, // Check every 5 seconds by default
		onChange:   onChange,
		stopCh:     make(chan struct{}),
	}
}

// Start begins watching the configuration file for changes
func (w *ConfigWatcher) Start(ctx context.Context) error {
	// Get initial file info
	info, err := os.Stat(w.configPath)
	if err != nil {
		return log.Errorf("failed to stat config file: %v", err)
	}
	w.lastMod = info.ModTime()

	w.wg.Add(1)
	go w.watchLoop(ctx)
	log.Info("Config watcher started for %s", w.configPath)
	return nil
}

// Stop stops watching the configuration file
func (w *ConfigWatcher) Stop() {
	w.mu.Lock()
	defer w.mu.Unlock()

	select {
	case <-w.stopCh:
		// Already stopped
		return
	default:
		close(w.stopCh)
	}

	w.wg.Wait()
	log.Info("Config watcher stopped")
}

// watchLoop periodically checks the configuration file for changes
func (w *ConfigWatcher) watchLoop(ctx context.Context) {
	defer w.wg.Done()

	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			w.checkForChanges()
		case <-ctx.Done():
			log.Info("Config watcher stopping due to context cancellation")
			return
		case <-w.stopCh:
			log.Info("Config watcher stopping due to stop signal")
			return
		}
	}
}

// checkForChanges checks if the configuration file has been modified
func (w *ConfigWatcher) checkForChanges() {
	w.mu.Lock()
	defer w.mu.Unlock()

	info, err := os.Stat(w.configPath)
	if err != nil {
		log.Warn("Failed to stat config file: %v", err)
		return
	}

	if info.ModTime().After(w.lastMod) {
		log.Info("Configuration file changed, reloading")

		// Load the new configuration
		newConfig, err := LoadConfig(w.configPath)
		if err != nil {
			log.Error("Failed to load new configuration: %v", err)
			return
		}

		// Update last modified time
		w.lastMod = info.ModTime()

		// Call the onChange callback with the new configuration
		if w.onChange != nil {
			w.onChange(newConfig)
		}
	}
}

// SetInterval sets the interval for checking file changes
func (w *ConfigWatcher) SetInterval(interval time.Duration) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.interval = interval
}
