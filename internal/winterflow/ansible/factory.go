package ansible

import (
	"winterflow-agent/internal/config"
	pkgansible "winterflow-agent/pkg/ansible"
)

func NewAnsibleClient(config *config.Config) pkgansible.Client {
	return pkgansible.NewClient(&pkgansible.Config{
		AnsiblePath:     config.AnsiblePath,
		AnsibleAppsPath: config.AnsibleAppsPath,
		AnsibleLogsPath: config.AnsiblePath,
		AppsPath:        config.AppsPath,
	})
}
