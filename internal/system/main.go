// Package system provides functionality for system-level operations and device management
package system

import (
	"winterflow-agent/internal/config"
	internaldevice "winterflow-agent/internal/system/device"
	internalsystem "winterflow-agent/internal/system/system"
)

// UpdateSystem updates the system packages
func UpdateSystem() error {
	return internalsystem.UpdateSystem()
}

// InstallRequiredPackages installs necessary system packages
func InstallRequiredPackages() error {
	return internalsystem.InstallRequiredPackages()
}

// ManagePlaybooksRepo ensures the winterflow playbooks repository is cloned and up to date
func ManagePlaybooksRepo(cfg *config.Config) error {
	return internalsystem.ManagePlaybooksRepo(cfg)
}

func GetDeviceID() (string, error) {
	return internaldevice.GetDeviceID()
}
