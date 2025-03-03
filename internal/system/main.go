// Package system provides functionality for system-level operations and device management
package system

import (
	internaldevice "winterflow-agent/internal/system/device"
	internalsystem "winterflow-agent/internal/system/system"
)

// FetchPackagesUpdates fetches the system packages updates
func FetchPackagesUpdates() error {
	return internalsystem.FetchPackagesUpdates()
}

// InstallRequiredPackages installs necessary system packages
func InstallRequiredPackages() error {
	return internalsystem.InstallRequiredPackages()
}

// DownloadPlaybooks ensures the winterflow playbooks repository is cloned and up to date
func DownloadPlaybooks(playbooksPath string) error {
	return internalsystem.DownloadPlaybooks(playbooksPath)
}

func GetDeviceID() (string, error) {
	return internaldevice.GetDeviceID()
}
