package ansible

import (
	"sync"
	"winterflow-agent/internal/application/config"
	"winterflow-agent/internal/domain/repository"
	pkgansible "winterflow-agent/pkg/ansible"
	log "winterflow-agent/pkg/log"
)

// ansibleRepository implements the RunnerRepository interface
type ansibleRepository struct {
	client pkgansible.Client
	mu     sync.RWMutex
	config *config.Config
}

// NewRepository creates a new Ansible repository
func NewRepository(config *config.Config) repository.RunnerRepository {
	client := NewAnsibleClient(config)
	return &ansibleRepository{
		client: client,
		config: config,
	}
}

func (r *ansibleRepository) GetRunner() pkgansible.Client {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.client
}

func (r *ansibleRepository) DeployIngress() {
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

func (r *ansibleRepository) DeployApp(appID, appVersion string) pkgansible.Result {
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

func (r *ansibleRepository) RestartApp(appID, appVersion string) pkgansible.Result {
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

func (r *ansibleRepository) StopApp(appID string) pkgansible.Result {
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

func (r *ansibleRepository) UpdateApp(appID string) pkgansible.Result {
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

func (r *ansibleRepository) DeleteApp(appID string) pkgansible.Result {
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

func (r *ansibleRepository) GenerateAppsStatus(statusOutputPath string) pkgansible.Result {
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
