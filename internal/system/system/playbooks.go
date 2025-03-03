package system

import (
	"fmt"
	"os"
	"path/filepath"
)

const (
	// PlaybooksRepoURL is the URL of the winterflow playbooks repository
	PlaybooksRepoURL = "https://github.com/winterflowio/winterflow-playbooks"
)

// DownloadPlaybooks ensures the winterflow-playbooks repository is cloned and up to date
func DownloadPlaybooks(playbooksPath string) error {
	if err := checkOS(); err != nil {
		return err
	}

	// Create playbooks directory if it doesn't exist
	if err := os.MkdirAll(filepath.Dir(playbooksPath), 0755); err != nil {
		return fmt.Errorf("failed to create playbooks directory: %w", err)
	}

	// Check if repository directory exists
	if _, err := os.Stat(playbooksPath); os.IsNotExist(err) {
		// Clone the repository
		err = runCommand("git", "clone", PlaybooksRepoURL, playbooksPath)
		if err != nil {
			return fmt.Errorf("failed to clone repository: %w", err)
		}
	} else {
		// Change to repository directory
		if err := os.Chdir(playbooksPath); err != nil {
			return fmt.Errorf("failed to change directory: %w", err)
		}
		// Pull latest changes
		if err := runCommand("git", "pull"); err != nil {
			return fmt.Errorf("failed to pull repository updates: %w", err)
		}
	}

	return nil
}
