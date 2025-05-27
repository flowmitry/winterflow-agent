package ansible

import (
	"winterflow-agent/internal/config"
	pkgansible "winterflow-agent/pkg/ansible"
)

func NewAnsibleClient(config *config.Config) pkgansible.Client {
	return pkgansible.NewClient(&pkgansible.Config{
		Orchestrator:                   config.GetOrchestrator(),
		AnsiblePath:                    config.GetAnsiblePath(),
		AnsibleAppsRolesPath:           config.GetAnsibleAppsRolesPath(),
		AnsibleLogsPath:                config.GetLogsPath(),
		AnsibleAppsRolesCurrentVersion: config.GetAnsibleAppRoleCurrentVersionFolder(),
	})
}
