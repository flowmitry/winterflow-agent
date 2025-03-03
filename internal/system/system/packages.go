package system

import (
	"fmt"
	"strings"
)

const (
	// RequiredPackages is a list of packages that need to be installed
	RequiredPackages = "ansible git"
)

// FetchPackagesUpdates updates system packages using apt package manager
func FetchPackagesUpdates() error {
	if err := checkOS(); err != nil {
		return err
	}

	// Update package lists
	if err := runCommand("sudo", "apt-get", "update"); err != nil {
		return fmt.Errorf("failed to update package lists: %w", err)
	}

	return nil
}

// InstallRequiredPackages installs necessary packages like Ansible and Git
func InstallRequiredPackages() error {
	if err := checkOS(); err != nil {
		return err
	}

	packages := strings.Fields(RequiredPackages)
	for _, pkg := range packages {
		if err := runCommand("sudo", "apt-get", "install", "-y", pkg); err != nil {
			return fmt.Errorf("failed to install %s: %w", pkg, err)
		}
	}

	return nil
}
