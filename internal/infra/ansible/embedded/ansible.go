package embedded

import (
	"io/fs"
	log "winterflow-agent/pkg/log"

	"winterflow-agent/internal/application/config"
	"winterflow-agent/internal/application/version"
	"winterflow-agent/pkg/embedded"
)

// Manager handles ansible-related operations
type Manager struct {
	embeddedManager *embedded.Manager
}

// NewManager creates a new Ansible manager
func NewManager(embeddedFS fs.FS, configPath string) *Manager {
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	return &Manager{
		embeddedManager: embedded.NewManager(embeddedFS, cfg.GetAnsibleFolder(), version.GetVersion(), []string{
			"inventory/defaults.yml",
			"playbooks",
			"roles",
			"ansible.cfg",
			"apps_roles/README.md",
		}),
	}
}

// SyncFiles ensures the ansible directory is up to date with the embedded files
func (m *Manager) SyncFiles() error {
	return m.embeddedManager.SyncFiles()
}
