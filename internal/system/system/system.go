package system

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"winterflow-agent/internal/config"
)

const (
	// ErrUnsupportedOS is returned when the operating system is not Linux
	ErrUnsupportedOS = "this agent only supports Linux operating systems"
	// PlaybooksRepoURL is the URL of the winterflow playbooks repository
	PlaybooksRepoURL = "https://github.com/winterflowio/winterflow-playbooks"
)

// checkOS verifies that the current operating system is Linux
func checkOS() error {
	if runtime.GOOS != "linux" {
		return fmt.Errorf(ErrUnsupportedOS)
	}
	return nil
}

// UpdateSystem updates system packages using apt package manager
func UpdateSystem() error {
	if err := checkOS(); err != nil {
		return err
	}

	// Update package lists
	if err := runCommand("sudo", "apt-get", "update"); err != nil {
		return fmt.Errorf("failed to update package lists: %w", err)
	}

	// Upgrade packages
	if err := runCommand("sudo", "apt-get", "-y", "upgrade"); err != nil {
		return fmt.Errorf("failed to upgrade packages: %w", err)
	}

	return nil
}

// InstallRequiredPackages installs necessary packages like Ansible and Git
func InstallRequiredPackages() error {
	if err := checkOS(); err != nil {
		return err
	}

	// Install Ansible
	if err := runCommand("sudo", "apt-get", "install", "-y", "ansible"); err != nil {
		return fmt.Errorf("failed to install Ansible: %w", err)
	}

	// Install Git
	if err := runCommand("sudo", "apt-get", "install", "-y", "git"); err != nil {
		return fmt.Errorf("failed to install Git: %w", err)
	}

	return nil
}

// ManagePlaybooksRepo ensures the winterflow-playbooks repository is cloned and up to date
func ManagePlaybooksRepo(cfg *config.Config) error {
	if err := checkOS(); err != nil {
		return err
	}

	// Create playbooks directory if it doesn't exist
	if err := os.MkdirAll(filepath.Dir(cfg.PlaybooksPath), 0755); err != nil {
		return fmt.Errorf("failed to create playbooks directory: %w", err)
	}

	// Check if repository directory exists
	if _, err := os.Stat(cfg.PlaybooksPath); os.IsNotExist(err) {
		// Clone the repository
		err = runCommand("git", "clone", PlaybooksRepoURL, cfg.PlaybooksPath)
		if err != nil {
			return fmt.Errorf("failed to clone repository: %w", err)
		}
	} else {
		// Change to repository directory
		if err := os.Chdir(cfg.PlaybooksPath); err != nil {
			return fmt.Errorf("failed to change directory: %w", err)
		}
		// Pull latest changes
		if err := runCommand("git", "pull"); err != nil {
			return fmt.Errorf("failed to pull repository updates: %w", err)
		}
	}

	return nil
}

// runCommand executes a command and returns any error
func runCommand(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
