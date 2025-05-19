package ansible

import (
	"winterflow-agent/internal/config"
	configconst "winterflow-agent/internal/config"
	pkgansible "winterflow-agent/pkg/ansible"
)

func NewAnsibleClient(config *config.Config) pkgansible.Client {
	return pkgansible.NewClient(&pkgansible.Config{
		AnsiblePath:                    config.AnsiblePath,
		AnsibleAppsRolesPath:           config.GetAnsibleAppsRolesPath(),
		AnsibleAppsRolesCurrentVersion: configconst.AnsibleAppsRolesCurrentVersionFolder,
		AnsibleLogsPath:                config.LogsPath,
	})
}
