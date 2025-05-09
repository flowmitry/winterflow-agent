package ansible

import (
	"io/fs"
	log "winterflow-agent/pkg/log"

	"winterflow-agent/internal/agent"
	"winterflow-agent/pkg/embedded"
)

// Manager handles ansible-related operations
type Manager struct {
	embeddedManager *embedded.Manager
}

// NewManager creates a new Ansible manager
func NewManager(embeddedFS fs.FS, configPath string) *Manager {
	cfg, err := agent.LoadConfig(configPath)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	return &Manager{
		embeddedManager: embedded.NewManager(embeddedFS, cfg.AnsiblePath, agent.GetVersion(), []string{
			"inventory/defaults.yml",
			"playbooks",
			"roles",
			"ansible.cfg",
		}),
	}
}

// SyncAnsibleFiles ensures the ansible directory is up to date with the embedded files
func (m *Manager) SyncAnsibleFiles() error {
	log.Printf("Syncing ansible files")
	return m.embeddedManager.SyncFiles()
}
