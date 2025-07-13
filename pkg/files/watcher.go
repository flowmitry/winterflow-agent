package files

import (
	"context"
	"os"
	"sync"
	"time"
	"winterflow-agent/pkg/log"
)

// FileWatcher watches a file for changes and calls a callback when modified
type FileWatcher struct {
	filePath string
	lastMod  time.Time
	interval time.Duration
	onChange func(string)
	stopCh   chan struct{}
	mu       sync.Mutex
	wg       sync.WaitGroup
}

// NewFileWatcher creates a new file watcher
func NewFileWatcher(filePath string, onChange func(string)) *FileWatcher {
	return &FileWatcher{
		filePath: filePath,
		interval: 5 * time.Second, // Check every 5 seconds by default
		onChange: onChange,
		stopCh:   make(chan struct{}),
	}
}

// Start begins watching the file for changes
func (w *FileWatcher) Start(ctx context.Context) error {
	// Get initial file info
	info, err := os.Stat(w.filePath)
	if err != nil {
		return log.Errorf("failed to stat file: %v", err)
	}
	w.lastMod = info.ModTime()

	w.wg.Add(1)
	go w.watchLoop(ctx)
	log.Info("File watcher started for %s", w.filePath)
	return nil
}

// Stop stops watching the file
func (w *FileWatcher) Stop() {
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
	log.Info("File watcher stopped")
}

// watchLoop periodically checks the file for changes
func (w *FileWatcher) watchLoop(ctx context.Context) {
	defer w.wg.Done()

	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			w.checkForChanges()
		case <-ctx.Done():
			log.Info("File watcher stopping due to context cancellation")
			return
		case <-w.stopCh:
			log.Info("File watcher stopping due to stop signal")
			return
		}
	}
}

// checkForChanges checks if the file has been modified
func (w *FileWatcher) checkForChanges() {
	w.mu.Lock()
	defer w.mu.Unlock()

	info, err := os.Stat(w.filePath)
	if err != nil {
		log.Warn("Failed to stat file: %v", err)
		return
	}

	if info.ModTime().After(w.lastMod) {
		log.Info("File changed: %s", w.filePath)

		// Update last modified time
		w.lastMod = info.ModTime()

		// Call the onChange callback with the file path
		if w.onChange != nil {
			w.onChange(w.filePath)
		}
	}
}

// SetInterval sets the interval for checking file changes
func (w *FileWatcher) SetInterval(interval time.Duration) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.interval = interval
}

// GetFilePath returns the path of the file being watched
func (w *FileWatcher) GetFilePath() string {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.filePath
}
