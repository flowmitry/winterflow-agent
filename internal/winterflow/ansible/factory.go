package ansible

import (
	"winterflow-agent/internal/config"
	pkgansible "winterflow-agent/pkg/ansible"
)

func NewAnsibleClient(config *config.Config) pkgansible.Client {
	return pkgansible.NewClient(&pkgansible.Config{
		AnsiblePath:                    config.GetAnsiblePath(),
		AnsibleAppsRolesPath:           config.GetAnsibleAppsRolesPath(),
		AnsibleAppsRolesCurrentVersion: config.GetAnsibleAppRoleCurrentVersionFolder(),
		AnsibleLogsPath:                config.LogsPath,
	})
}
