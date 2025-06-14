package ansible

import (
	"sync"
	"winterflow-agent/internal/config"
	pkgansible "winterflow-agent/pkg/ansible"
	log "winterflow-agent/pkg/log"
)

// Repository is an interface for managing Ansible runners
type Repository interface {
	// GetRunner returns an Ansible client for running playbooks
	GetRunner() pkgansible.Client

	// DeployIngress runs the initial configuration playbook
	DeployIngress()

	// DeployApp deploys an application with the specified ID and version
	DeployApp(appID, appVersion string) pkgansible.Result

	// StopApp stops the application specified by the given app ID and returns the result of the operation.
	StopApp(appID string) pkgansible.Result

	// RestartApp restarts the specified application by its app ID and version and returns the result of the operation.
	RestartApp(appID, appVersion string) pkgansible.Result

	// UpdateApp updates the specified application by its app ID and version and returns the result of the operation.
	UpdateApp(appID string) pkgansible.Result

	// DeleteApp removes an application identified by the provided appID and returns the result of the operation.
	DeleteApp(appID string) pkgansible.Result

	// GenerateAppsStatus generate json files with the status of all applications
	GenerateAppsStatus(statusOutputPath string) pkgansible.Result
}

// repository implements the Repository interface
type repository struct {
	client pkgansible.Client
	mu     sync.RWMutex
	config *config.Config
}

// NewRepository creates a new Ansible repository
func NewRepository(config *config.Config) *repository {
	client := NewAnsibleClient(config)
	return &repository{
		client: client,
		config: config,
	}
}

func (r *repository) GetRunner() pkgansible.Client {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.client
}

func (r *repository) DeployIngress() {
	log.Debug("Running ingress configuration playbook")
	if r.config.IsFeatureEnabled(config.FeatureIngressDisabled) {
		log.Info("Initial configuration skipped as ingress disabled")
	} else {
		res := r.client.RunSync(pkgansible.Command{
			Playbook: "ingress/stop_ingress.yml",
		})
		if res.Error != nil {
			log.Warn("Error running ingress/stop_ingress.yml playbook", res.Error)
		}

		res = r.client.RunSync(pkgansible.Command{
			Playbook: "ingress/deploy_ingress.yml",
		})
		if res.Error != nil {
			log.Error("Error running ingress/deploy_ingress.yml playbook", res.Error)
		}
	}
}

func (r *repository) DeployApp(appID, appVersion string) pkgansible.Result {
	log.Debug("Deploying application with ID %s and version %s", appID, appVersion)
	env := map[string]string{
		"app_id":      appID,
		"app_version": appVersion,
	}
	cmd := pkgansible.Command{
		Playbook: "apps/deploy_app.yml",
		Env:      env,
	}
	return r.client.RunSync(cmd)
}

func (r *repository) RestartApp(appID, appVersion string) pkgansible.Result {
	log.Debug("Restarting application with ID %s and version %s", appID, appVersion)
	env := map[string]string{
		"app_id":      appID,
		"app_version": appVersion,
	}
	cmd := pkgansible.Command{
		Playbook: "apps/restart_app.yml",
		Env:      env,
	}
	return r.client.RunSync(cmd)
}

func (r *repository) StopApp(appID string) pkgansible.Result {
	log.Debug("Stoping application with ID %s", appID)
	env := map[string]string{
		"app_id": appID,
	}
	cmd := pkgansible.Command{
		Playbook: "apps/stop_app.yml",
		Env:      env,
	}
	return r.client.RunSync(cmd)
}

func (r *repository) UpdateApp(appID string) pkgansible.Result {
	log.Debug("Updating application with ID %s", appID)
	env := map[string]string{
		"app_id": appID,
	}
	cmd := pkgansible.Command{
		Playbook: "apps/update_app.yml",
		Env:      env,
	}
	return r.client.RunSync(cmd)
}

func (r *repository) DeleteApp(appID string) pkgansible.Result {
	log.Debug("Deleting application with ID %s and version %s", appID)
	env := map[string]string{
		"app_id": appID,
	}
	cmd := pkgansible.Command{
		Playbook: "apps/delete_app.yml",
		Env:      env,
	}
	return r.client.RunSync(cmd)
}

func (r *repository) GenerateAppsStatus(statusOutputPath string) pkgansible.Result {
	log.Debug("Getting status of all applications")
	env := map[string]string{
		"apps_status_output_path": statusOutputPath,
	}
	cmd := pkgansible.Command{
		Playbook: "apps/get_apps_status.yml",
		Env:      env,
	}
	return r.client.RunSync(cmd)
}
