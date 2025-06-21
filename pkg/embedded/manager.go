package embedded

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// Manager handles embedded files operations
// It extracts embedded files into a target directory. If the directory already exists the
// embedded files are overwritten to guarantee they are up-to-date.
//
// The manager purposefully avoids any version tracking logic – callers control when the
// extraction happens. This greatly simplifies the behaviour and eliminates the need for
// auxiliary files (like version.txt) or complex "paths to remove" semantics.

type Manager struct {
	embeddedFS fs.FS
	targetDir  string
}

// NewManager creates a new embedded files manager.
func NewManager(embeddedFS fs.FS, targetDir string) *Manager {
	return &Manager{
		embeddedFS: embeddedFS,
		targetDir:  targetDir,
	}
}

// SyncFiles extracts the embedded files into the target directory, overwriting any existing
// files. If the directory does not yet exist it will be created.
func (m *Manager) SyncFiles() error {
	if err := m.extractFiles(); err != nil {
		return fmt.Errorf("failed to extract embedded files: %w", err)
	}
	return nil
}

// extractFiles walks through the embedded FS and writes every entry into the target directory.
func (m *Manager) extractFiles() error {
	// Ensure the target directory exists
	if err := os.MkdirAll(m.targetDir, 0755); err != nil {
		return fmt.Errorf("failed to create target directory: %w", err)
	}

	// Walk all files inside the embedded filesystem and copy them over.
	err := fs.WalkDir(m.embeddedFS, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Skip root directory
		if path == "." {
			return nil
		}

		targetPath := filepath.Join(m.targetDir, path)

		if d.IsDir() {
			// Create directory
			return os.MkdirAll(targetPath, 0755)
		}

		// Read embedded file
		data, err := fs.ReadFile(m.embeddedFS, path)
		if err != nil {
			return fmt.Errorf("failed to read embedded file %s: %w", path, err)
		}

		// Create parent directories if they don't exist
		parentDir := filepath.Dir(targetPath)
		if err := os.MkdirAll(parentDir, 0755); err != nil {
			return fmt.Errorf("failed to create parent directory for %s: %w", targetPath, err)
		}

		// Write file
		if err := os.WriteFile(targetPath, data, 0644); err != nil {
			return fmt.Errorf("failed to write file %s: %w", targetPath, err)
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to extract files: %w", err)
	}

	return nil
}
