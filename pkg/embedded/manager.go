package embedded

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

const (
	// VersionFile is the name of the version file
	VersionFile = "version.txt"
)

// Manager handles embedded files operations
type Manager struct {
	embeddedFS fs.FS
	targetDir  string
	version    string
}

// NewManager creates a new embedded files manager
func NewManager(embeddedFS fs.FS, targetDir string, version string) *Manager {
	return &Manager{
		embeddedFS: embeddedFS,
		targetDir:  targetDir,
		version:    version,
	}
}

// SyncFiles ensures the target directory is up to date with the embedded files
func (m *Manager) SyncFiles() error {
	// Check if target directory exists
	exists, err := m.checkTargetDir()
	if err != nil {
		return fmt.Errorf("failed to check target directory: %w", err)
	}

	// Check version if directory exists
	if exists {
		needsUpdate, err := m.checkVersion()
		if err != nil {
			return fmt.Errorf("failed to check version: %w", err)
		}
		if !needsUpdate {
			return nil
		}
	}

	// Extract and replace files
	if err := m.extractAndReplace(); err != nil {
		return fmt.Errorf("failed to extract and replace files: %w", err)
	}

	return nil
}

// checkTargetDir checks if the target directory exists
func (m *Manager) checkTargetDir() (bool, error) {
	_, err := os.Stat(m.targetDir)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

// checkVersion checks if the current version matches the version in version.txt
func (m *Manager) checkVersion() (bool, error) {
	versionPath := filepath.Join(m.targetDir, VersionFile)
	data, err := os.ReadFile(versionPath)
	if err != nil {
		if os.IsNotExist(err) {
			return true, nil
		}
		return false, err
	}

	currentVersion := strings.TrimSpace(string(data))
	return currentVersion != m.version, nil
}

// extractAndReplace extracts embedded files and replaces existing ones
func (m *Manager) extractAndReplace() error {
	// Remove existing directory if it exists
	if err := os.RemoveAll(m.targetDir); err != nil {
		return fmt.Errorf("failed to remove existing target directory: %w", err)
	}

	// Create new directory
	if err := os.MkdirAll(m.targetDir, 0755); err != nil {
		return fmt.Errorf("failed to create target directory: %w", err)
	}

	// Walk through embedded files and extract them
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
			return os.MkdirAll(targetPath, 0755)
		}

		// Read embedded file
		data, err := fs.ReadFile(m.embeddedFS, path)
		if err != nil {
			return fmt.Errorf("failed to read embedded file %s: %w", path, err)
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

	// Write version file
	versionPath := filepath.Join(m.targetDir, VersionFile)
	if err := os.WriteFile(versionPath, []byte(m.version), 0644); err != nil {
		return fmt.Errorf("failed to write version file: %w", err)
	}

	return nil
}
